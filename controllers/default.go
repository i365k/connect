package controllers

import (
	"encoding/json"
	"github.com/astaxie/beego/cache"
	_ "github.com/astaxie/beego/cache/redis"
	_ "github.com/gomodule/redigo/redis"
	"github.com/gomodule/redigo/redis"
	"github.com/astaxie/beego/logs"
	"github.com/astaxie/beego/orm"
	"strconv"
	"github.com/astaxie/beego"
	"connect/utils"
	"connect/models"
	"github.com/medivhzhan/weapp"
	"github.com/goEncrypt"
	"time"
	"encoding/hex"
	"fmt"
	"math/rand"
)

//创建一个结构体继承beego
type Controller struct {
	beego.Controller
}

type New struct {
	Name      string `json:"name"`
	Url       string `json:"url"`
	GetStatus string `json:"getstatus"`
}

type userPai struct {
	Name       string `json:"name"`
	Url        string `json:"url"`
	Checkpoint string `json:"checkpoint"`
	Ranking    string `json:"ranking"`
	MaxNumber  string `json:"max_number"`
}

//健康检查
func (this *Controller) Ping() {
	//打印日志
	logs.SetLogger(logs.AdapterFile, `{"filename":"./logs/api/ping.log"}`)

	//定义反回数据
	response := map[string]string{
	}

	response["errno"] = utils.RECODE_OK
	response["errmsg"] = utils.RecodeText(response["errno"])
	this.Data["json"] = response
	//调用sercejson方法返回json数据
	this.ServeJSON()
	return
}

//用户登陆
func (this *Controller) Login() {
	//打印日志
	//logs.SetLogger(logs.AdapterFile, `{"filename":"./logs/connect.log"}`)
	logs.Info("Login 登陆服务 /api/connect/login")
	logs.SetLevel(1)

	//定义反回数据
	response := map[string]string{
	}

	//接收请求参数
	var request = make(map[string]interface{})
	if err := json.NewDecoder(this.Ctx.Request.Body).Decode(&request); err != nil {
		return
	}
	tempuserinfo := request["userInfo"].(map[string]interface{})
	nickName := tempuserinfo["nickName"]
	logs.Info(nickName)
	gender := tempuserinfo["gender"]
	logs.Info(gender)
	language := tempuserinfo["language"]
	logs.Info(language)
	city := tempuserinfo["city"]
	logs.Info(city)
	province := tempuserinfo["province"]
	logs.Info(province)
	country := tempuserinfo["country"]
	logs.Info(country)
	avatarUrl := tempuserinfo["avatarUrl"]
	logs.Info(avatarUrl)

	//请求微信接口获取session_key和openid
	appID := "wx83175b04f104d2ef"
	secret := "2ca7431708347305ce46644a26060122"
	code := request["code"].(string)
	//调用微信接口获取数据
	res, err := weapp.Login(appID, secret, code)
	if err != nil {
		logs.Info(err)
		logs.Info("code验证失败")
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"])
		this.Data["json"] = response
		this.ServeJSON()
		return
	}

	ssk := res.SessionKey
	//调用sha256方法将session_key哈希得到skey
	skey := goEncrypt.GetStringHash256(ssk)
	//将从微信获取到的opid哈希处理
	hxopid := utils.GetMd5String(res.OpenID)

	//配置缓存参数
	redis_conf := map[string]string{
		"key":      utils.G_server_name,
		"conn":     utils.G_redis_addr + ":" + utils.G_redis_port,
		"dbNum":    utils.G_redis_dbnum,
		"password": utils.G_redis_password,
	}
	logs.Info(redis_conf)
	//将map进行转化成为json
	redis_conf_js, _ := json.Marshal(redis_conf)
	//创建redis句柄
	bm, err := cache.NewCache("redis", string(redis_conf_js))
	if err != nil {
		logs.Info("redis连接失败", err)
		response["errno"] = utils.RECODE_DBERR
		response["errmsg"] = utils.RecodeText(response["errno"])
		this.Data["json"] = response
		this.ServeJSON()
		return
	}

	o := orm.NewOrm()
	//判断是否新用户
	var usertemp models.User
	err = o.QueryTable("user").Filter("hxopid", hxopid).One(&usertemp)
	if err == orm.ErrNoRows {
		//判断为新用户
		logs.Info("新用户登陆")
		//插入用户数据
		usertemp.OpenId = res.OpenID
		usertemp.Hxopid = hxopid
		usertemp.Name = nickName.(string)
		usertemp.Gender = strconv.Itoa(int(gender.(float64)))
		usertemp.Session_key = res.SessionKey
		usertemp.Skey = skey
		usertemp.Avatar_url = avatarUrl.(string)
		usertemp.Landing_time = strconv.Itoa(int(time.Now().Unix()))
		usertemp.MaxLevel = 0
		usertemp.MaxConfig = "1"
		usertemp.MiniConfig = "1"
		usertemp.UserMoney = "0"
		usertemp.Package = "3"
		usertemp.SignNumber = "0"
		num, err := o.Insert(&usertemp)
		if err != nil {
			logs.Info("插入新用户失败", err)
		}
		logs.Info("插入新用户成功", num)

		var user models.User
		o.QueryTable("user").Filter("hxopid", hxopid).One(&user)

		//将skey对应的openid写入缓存
		bm.Put(skey, hxopid, time.Second*36000)
		if err != nil {
			logs.Info("存入redis失败", err)
			response["errno"] = utils.RECODE_DBERR
			response["errmsg"] = utils.RecodeText(response["errno"])
			this.Data["json"] = response
			this.ServeJSON()
			return
		}
		logs.Info("opid插入redis成功")

		//判断是否为受邀请用户
		shareskey := request["shareskey"].(string)
		if len(shareskey) > 15 {
			logs.Info(shareskey)
			//从缓存中取出哈希过的opid
			opid := bm.Get(shareskey)
			if opid == nil {
				logs.Info("超过分享时间,判定为普通用户登陆")
				//rsp.Errno = utils.RECODE_PARAMERR
				//rsp.Errmsg = utils.RecodeText(rsp.Errno)
				//return nil
			}
			logs.Info("获取分享人信息成功,判定为邀请用户登陆")
			//将取出的数据转换为string
			opidtemp, _ := redis.String(opid, nil)

			var user models.User
			o.QueryTable("user").Filter("hxopid", opidtemp).One(&user)
			invitations := models.Invitation{
				Invitation_OpenId:     hxopid,
				Invitation_Name:       nickName.(string),
				Invitation_Avatar_url: avatarUrl.(string),
				GetStatus:             "true",
				User:                  &user,
			}
			o.Insert(&invitations)
			logs.Info("插入邀请新人表成功")
		}

		//返回数据
		response["skey"] = skey
		response["errno"] = utils.RECODE_OK
		response["errmsg"] = utils.RecodeText(response["errno"])
		this.Data["json"] = response
		//调用sercejson方法返回json数据
		this.ServeJSON()
		return
	}
	//判断为老用户
	logs.Info("老用户登陆", usertemp.Name)
	//更新用户数据
	usertemp.Skey = skey
	usertemp.Name = nickName.(string)
	usertemp.Gender = strconv.Itoa(int(gender.(float64)))
	usertemp.Avatar_url = avatarUrl.(string)
	usertemp.Landing_time = strconv.Itoa(int(time.Now().Unix()))
	num, err := o.Update(&usertemp)
	if err != nil {
		logs.Info("老用户更新数据失败", err)
	}
	logs.Info(num)

	//将skey对应的openid写入缓存
	bm.Put(skey, hxopid, time.Second*36000)
	if err != nil {
		logs.Info("存入redis失败", err)
		response["errno"] = utils.RECODE_DBERR
		response["errmsg"] = utils.RecodeText(response["errno"])
		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	logs.Info("opid插入redis成功")

	//返回数据
	response["skey"] = skey
	response["errno"] = utils.RECODE_OK
	response["errmsg"] = utils.RecodeText(response["errno"])
	this.Data["json"] = response
	//调用sercejson方法返回json数据
	this.ServeJSON()
	return
}

//拉取邀请新人列表
func (this *Controller) PullNew() {
	//打印日志
	//logs.SetLogger(logs.AdapterFile, `{"filename":"./logs/connect.log"}`)
	logs.Info("PullNew 拉取邀请新人列表 /api/connect/pullnew")

	//接收请求参数
	var request map[string]interface{}
	if err := json.NewDecoder(this.Ctx.Request.Body).Decode(&request); err != nil {
		return
	}
	//定义反回数据
	response := map[string]interface{}{

	}
	//判断请求参数
	if _, ok := request["skey"].(string); ok != true {
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"].(string))

		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	if _, ok := request["url"].(string); ok != true {
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"].(string))

		this.Data["json"] = response
		this.ServeJSON()
		return
	}

	//准备请求参数
	Skey := request["skey"].(string)
	Url := utils.Decrypt(request["url"].(string))

	//配置缓存参数
	var redis_conf = map[string]string{
		"key": utils.G_server_name,
		//127.0.0.1:6379
		"conn":     utils.G_redis_addr + ":" + utils.G_redis_port,
		"dbNum":    utils.G_redis_dbnum,
		"password": utils.G_redis_password,
	}
	//将map进行转化成为json
	var redis_conf_js, _ = json.Marshal(redis_conf)

	//创建redis句柄
	bm, err := cache.NewCache("redis", string(redis_conf_js))
	if err != nil {
		logs.Info("redis连接失败", err)
		response["errno"] = utils.RECODE_DBERR
		response["errmsg"] = utils.RecodeText(response["errno"].(string))
		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	logs.Info("redis连接成功")
	//将同步来的data存入redis
	//从缓存中取出哈希过的opid
	opid := bm.Get(Skey)
	if opid == nil {
		logs.Info("获取opid失败", err)
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"].(string))
		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	logs.Info("获取opid成功")
	//将取出的数据转换为string
	opidtemp, _ := redis.String(opid, nil)

	o := orm.NewOrm()

	var user models.User
	o.QueryTable("user").Filter("hxopid", opidtemp).One(&user)
	if len(Url) > 10 {
		var TInvitations []models.Invitation
		o.QueryTable("invitation").Filter("user_id__hxopid", opidtemp).All(&TInvitations)
		for _, v := range TInvitations {
			var TTInvitations models.Invitation
			if v.Invitation_Avatar_url == Url {
				o.QueryTable("invitation").Filter("id", v.Id).One(&TTInvitations)
				if err == orm.ErrNoRows {
					response["errno"] = utils.RECODE_DBERR
					response["errmsg"] = utils.RecodeText(response["errno"].(string))
					this.Data["json"] = response
					this.ServeJSON()
					return
				}
				TTInvitations.GetStatus = "false"
				num, err := o.Update(&TTInvitations)
				if err != nil {
					logs.Info("更新领取状态失败", err)
				}
				usqq, _ := strconv.Atoi(user.UserMoney)
				usqq += 100
				user.UserMoney = strconv.Itoa(usqq)
				num, err = o.Update(&user)
				if err != nil {
					logs.Info("更新用户金币失败", err)
				}
				logs.Info(num, "更新用户金币成功")
			}
		}
	}

	var Invitations []models.Invitation
	o.QueryTable("invitation").Filter("user_id__hxopid", opidtemp).All(&Invitations)

	logs.Info("查询数据成功")

	var Newlist []New
	for _, v := range Invitations {
		var templist New
		templist.Url = utils.Encrypt(v.Invitation_Avatar_url)
		templist.Name = utils.Encrypt(v.Invitation_Name)
		templist.GetStatus = utils.Encrypt(v.GetStatus)
		Newlist = append(Newlist, templist)
	}

	//返回数据
	//response["usermoney"] = user.UserMoney
	response["list"] = Newlist
	response["number"] = utils.Encrypt(user.UserMoney)
	response["errno"] = utils.RECODE_OK
	response["errmsg"] = utils.RecodeText(response["errno"].(string))
	this.Data["json"] = response
	//调用sercejson方法返回json数据
	this.ServeJSON()
	return
}

//服务器排名
func (this *Controller) Ranking() {
	//打印日志
	//logs.SetLogger(logs.AdapterFile, `{"filename":"./logs/connect.log"}`)
	logs.Info("Ranking 服务器排名 /api/connect/ranking")

	//接收请求参数
	var request map[string]interface{}
	if err := json.NewDecoder(this.Ctx.Request.Body).Decode(&request); err != nil {
		return
	}
	//定义反回数据
	response := map[string]interface{}{
	}
	//判断请求参数
	if _, ok := request["skey"].(string); ok != true {
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"].(string))

		this.Data["json"] = response
		this.ServeJSON()
		return
	}

	//准备请求参数
	Skey := request["skey"].(string)

	//配置缓存参数
	var redis_conf = map[string]string{
		"key": utils.G_server_name,
		//127.0.0.1:6379
		"conn":     utils.G_redis_addr + ":" + utils.G_redis_port,
		"dbNum":    utils.G_redis_dbnum,
		"password": utils.G_redis_password,
	}
	//将map进行转化成为json
	var redis_conf_js, _ = json.Marshal(redis_conf)

	//创建redis句柄
	bm, err := cache.NewCache("redis", string(redis_conf_js))
	if err != nil {
		logs.Info("redis连接失败", err)
		response["errno"] = utils.RECODE_DBERR
		response["errmsg"] = utils.RecodeText(response["errno"].(string))
		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	logs.Info("redis连接成功")

	//从缓存中取出哈希过的opid
	opid := bm.Get(Skey)
	if opid == nil {
		logs.Info("获取opid失败", err)
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"].(string))
		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	logs.Info("获取opid成功")
	//将取出的数据转换为string
	opidtemp, _ := redis.String(opid, nil)

	//这个地方可以查出排序后用户的下标,反给前端用来高亮
	logs.Info(opidtemp)
	o := orm.NewOrm()
	var users []models.User
	o.QueryTable("user").OrderBy("-max_level").All(&users)

	var user models.User
	o.QueryTable("user").Filter("hxopid", opidtemp).One(&user)
	logs.Info("查询数据成功")
	temp := 0
	var nus []userPai
	for k, v := range users {
		if v.Hxopid == opidtemp {
			temp = k + 1
		}
		var templist userPai
		templist.Name = utils.Encrypt(v.Name)
		templist.Url = utils.Encrypt(v.Avatar_url)
		templist.Checkpoint = utils.Encrypt(strconv.Itoa(v.MaxLevel))
		nus = append(nus, templist)
	}

	//返回数据
	response["list"] = nus
	response["Checkpoint"] = utils.Encrypt(strconv.Itoa(user.MaxLevel))
	response["Ranking"] = utils.Encrypt(strconv.Itoa(temp))
	response["errno"] = utils.RECODE_OK
	response["errmsg"] = utils.RecodeText(response["errno"].(string))
	this.Data["json"] = response
	//调用sercejson方法返回json数据
	this.ServeJSON()
	return
}

//限时模式服务器排名
func (this *Controller) TimeRanking() {
	//打印日志
	//logs.SetLogger(logs.AdapterFile, `{"filename":"./logs/connect.log"}`)
	logs.Info("Ranking 服务器排名 /api/connect/ranking")

	//接收请求参数
	var request map[string]interface{}
	if err := json.NewDecoder(this.Ctx.Request.Body).Decode(&request); err != nil {
		return
	}
	//定义反回数据
	response := map[string]interface{}{
	}
	//判断请求参数
	if _, ok := request["skey"].(string); ok != true {
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"].(string))

		this.Data["json"] = response
		this.ServeJSON()
		return
	}

	//准备请求参数
	Skey := request["skey"].(string)

	//配置缓存参数
	var redis_conf = map[string]string{
		"key": utils.G_server_name,
		//127.0.0.1:6379
		"conn":     utils.G_redis_addr + ":" + utils.G_redis_port,
		"dbNum":    utils.G_redis_dbnum,
		"password": utils.G_redis_password,
	}
	//将map进行转化成为json
	var redis_conf_js, _ = json.Marshal(redis_conf)

	//创建redis句柄
	bm, err := cache.NewCache("redis", string(redis_conf_js))
	if err != nil {
		logs.Info("redis连接失败", err)
		response["errno"] = utils.RECODE_DBERR
		response["errmsg"] = utils.RecodeText(response["errno"].(string))
		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	logs.Info("redis连接成功")

	//从缓存中取出哈希过的opid
	opid := bm.Get(Skey)
	if opid == nil {
		logs.Info("获取opid失败", err)
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"].(string))
		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	logs.Info("获取opid成功")
	//将取出的数据转换为string
	opidtemp, _ := redis.String(opid, nil)

	//这个地方可以查出排序后用户的下标,反给前端用来高亮
	logs.Info(opidtemp)
	o := orm.NewOrm()
	var users []models.User
	o.QueryTable("user").OrderBy("-time_level").All(&users)

	var user models.User
	o.QueryTable("user").Filter("hxopid", opidtemp).One(&user)
	logs.Info("查询数据成功")
	temp := 0
	var nus []userPai
	for k, v := range users {
		if v.Hxopid == opidtemp {
			temp = k + 1
			fmt.Println(temp, "++++++++++++++++", v.Hxopid)
		}
		var templist userPai
		templist.Name = utils.Encrypt(v.Name)
		templist.Url = utils.Encrypt(v.Avatar_url)
		templist.Checkpoint = utils.Encrypt(strconv.Itoa(v.TimeLevel))
		nus = append(nus, templist)
	}

	//返回数据
	response["list"] = nus
	response["Checkpoint"] = utils.Encrypt(strconv.Itoa(user.TimeLevel))
	response["Ranking"] = utils.Encrypt(strconv.Itoa(temp))
	response["errno"] = utils.RECODE_OK
	response["errmsg"] = utils.RecodeText(response["errno"].(string))
	this.Data["json"] = response
	//调用sercejson方法返回json数据
	this.ServeJSON()
	return
}

//服务器时间
func (this *Controller) NowTime() {
	//打印日志
	//logs.SetLogger(logs.AdapterFile, `{"filename":"./logs/connect.log"}`)
	logs.Info("NowTime 服务器时间 /api/connect/nowtime")

	//接收请求参数
	var request map[string]interface{}
	if err := json.NewDecoder(this.Ctx.Request.Body).Decode(&request); err != nil {
		return
	}
	//定义反回数据
	response := map[string]string{
	}
	//判断请求参数
	if _, ok := request["skey"].(string); !ok {
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"])
		this.Data["json"] = response
		this.ServeJSON()
		return
	}

	//准备请求参数
	Skey := request["skey"].(string)

	if len(Skey) < 10 {
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"])
		this.Data["json"] = response
		this.ServeJSON()
		return
	}

	//准备时间戳
	nowtime := time.Now().Unix()
	//准备加密
	Key := []byte("1234567887654321")
	ciphernowtime := goEncrypt.AesCBC_Encrypt([]byte(strconv.Itoa(int(nowtime))), Key)
	encodenowtime := hex.EncodeToString(ciphernowtime)

	//返回数据
	response["nowtime"] = encodenowtime
	response["errno"] = utils.RECODE_OK
	response["errmsg"] = utils.RecodeText(response["errno"])

	this.Data["json"] = response
	this.ServeJSON()
	return
}

//获取用户过关数据
func (this *Controller) GetCheckpoint() {
	//打印日志
	//logs.SetLogger(logs.AdapterFile, `{"filename":"./logs/connect.log"}`)
	logs.Info("GetMaxCheckpoint 获取最高关卡服务 /api/connect/getcheckpoint")

	//接收请求参数
	var request map[string]interface{}
	if err := json.NewDecoder(this.Ctx.Request.Body).Decode(&request); err != nil {
		return
	}
	//定义反回数据
	response := map[string]string{
	}
	//判断请求参数
	if _, ok := request["skey"].(string); ok != true {
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"])
		this.Data["json"] = response
		this.ServeJSON()
		return
	}

	//准备请求参数
	Skey := request["skey"].(string)

	//配置缓存参数
	var redis_conf = map[string]string{
		"key":      utils.G_server_name,
		"conn":     utils.G_redis_addr + ":" + utils.G_redis_port,
		"dbNum":    utils.G_redis_dbnum,
		"password": utils.G_redis_password,
	}
	//将map进行转化成为json
	var redis_conf_js, _ = json.Marshal(redis_conf)

	//创建redis句柄
	bm, err := cache.NewCache("redis", string(redis_conf_js))
	if err != nil {
		logs.Info("redis连接失败", err)
		response["errno"] = utils.RECODE_DBERR
		response["errmsg"] = utils.RecodeText(response["errno"])
		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	logs.Info("redis连接成功")
	//从缓存中取出哈希过的opid
	opid := bm.Get(Skey)
	if opid == nil {
		logs.Info("获取opid失败", err)
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"])
		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	logs.Info("获取opid成功")
	//将取出的数据转换为string
	opidtemp, _ := redis.String(opid, nil)

	o := orm.NewOrm()
	var user models.User
	err = o.QueryTable("user").Filter("hxopid", opidtemp).One(&user)
	if err != nil {
		logs.Info("查询用户数据失败")
	}
	logs.Info("查询数据成功")

	key1 := "max" + user.Hxopid
	var maxs []models.MaxConfig

	numbertemp_s := bm.Get(key1)
	if numbertemp_s == nil {
		logs.Info("新用户记录过关次数")
		o.QueryTable("max_config").All(&maxs)
		logs.Info("查询大关卡数据成功")

		maxsjson, _ := json.Marshal(&maxs)

		err = bm.Put(key1, maxsjson, time.Second*99999999)
		if err != nil {
			logs.Info("大关数放入redis失败", err)
		}
		logs.Info("大关数放入redis成功")

		//更新小关星数

		var minis [20]int

		for i := 1; i < 21; i++ {
			key := "mini" + strconv.Itoa(i) + user.Hxopid
			numbertemp_s := bm.Get(key)
			if numbertemp_s == nil {
				logs.Info("新用户生成小关数据")
				minisjson, _ := json.Marshal(&minis)
				err = bm.Put(key, minisjson, time.Second*99999999)
				if err != nil {
					logs.Info("大关数放入redis失败", err)
				}
				logs.Info("大关数放入redis成功", i)
			}
		}
	}

	//返回数据
	response["number"] = utils.Encrypt(strconv.Itoa(user.MaxLevel))
	response["maxnumber"] = utils.Encrypt(strconv.Itoa(user.TimeLevel))
	response["maxnumbera"] = utils.Encrypt(user.MaxConfig)
	response["mininumber"] = utils.Encrypt(user.MiniConfig)
	response["errno"] = utils.RECODE_OK
	response["errmsg"] = utils.RecodeText(response["errno"])
	this.Data["json"] = response
	//调用sercejson方法返回json数据
	this.ServeJSON()
	return
}

//获取大关数据
func (this *Controller) MaxConfig() {
	//打印日志
	//logs.SetLogger(logs.AdapterFile, `{"filename":"./logs/connect.log"}`)
	logs.Info("SetNowCheckpoint 记录过关次数 /api/connect/maxconfig")

	//接收请求参数
	var request map[string]interface{}
	if err := json.NewDecoder(this.Ctx.Request.Body).Decode(&request); err != nil {
		return
	}
	//定义反回数据
	response := map[string]interface{}{
	}
	//判断请求参数
	if _, ok := request["skey"].(string); ok != true {
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"].(string))
		this.Data["json"] = response
		this.ServeJSON()
		return
	}

	//准备请求参数
	Skey := request["skey"].(string)

	//配置缓存参数
	var redis_conf = map[string]string{
		"key": utils.G_server_name,

		"conn":     utils.G_redis_addr + ":" + utils.G_redis_port,
		"dbNum":    utils.G_redis_dbnum,
		"password": utils.G_redis_password,
	}
	//将map进行转化成为json
	var redis_conf_js, _ = json.Marshal(redis_conf)

	//创建redis句柄
	bm, err := cache.NewCache("redis", string(redis_conf_js))
	if err != nil {
		logs.Info("redis连接失败", err)
		response["errno"] = utils.RECODE_DBERR
		response["errmsg"] = utils.RecodeText(response["errno"].(string))
		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	logs.Info("redis连接成功")

	//从缓存中取出哈希过的opid
	opid := bm.Get(Skey)
	if opid == nil {
		logs.Info("获取opid失败", err)
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"].(string))
		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	logs.Info("获取opid成功")
	//将取出的数据转换为string
	opidtemp, _ := redis.String(opid, nil)

	o := orm.NewOrm()
	var user models.User
	o.QueryTable("user").Filter("hxopid", opidtemp).One(&user)
	logs.Info("查询用户数据成功")

	//更新大关数据
	key1 := "max" + user.Hxopid
	var maxs []models.MaxConfig

	numbertemp_s := bm.Get(key1)
	if numbertemp_s == nil {
	}
	//解码yp
	err = json.Unmarshal(numbertemp_s.([]byte), &maxs)
	if err != nil {
		logs.Info("解码失败")
	}
	logs.Info("获取过关次数成功")

	var cypMaxs []models.MaxConfig

	for _, v := range maxs {
		var cypMax models.MaxConfig
		cypMax.Proportion = utils.Encrypt(v.Proportion)
		cypMax.Mininumber = utils.Encrypt(v.Mininumber)
		cypMax.Status = utils.Encrypt(v.Status)
		cypMax.Name = utils.Encrypt(v.Name)
		cypMax.Code = utils.Encrypt(v.Code)
		cypMax.Url = utils.Encrypt(v.Url)
		cypMax.Bjurl = utils.Encrypt(v.Bjurl)
		cypMax.Usernumber = utils.Encrypt(v.Usernumber)
		cypMaxs = append(cypMaxs, cypMax)
	}

	response["errno"] = utils.RECODE_OK
	response["maxlist"] = cypMaxs
	response["errmsg"] = utils.RecodeText(response["errno"].(string))
	this.Data["json"] = response
	this.ServeJSON()
	return
}

//获取小关数据
func (this *Controller) MiniConfig() {
	//打印日志
	//logs.SetLogger(logs.AdapterFile, `{"filename":"./logs/connect.log"}`)
	logs.Info("SetNowCheckpoint 获取小关数据 /api/connect/miniconfig")

	//接收请求参数
	var request map[string]interface{}
	if err := json.NewDecoder(this.Ctx.Request.Body).Decode(&request); err != nil {
		return
	}
	//定义反回数据
	response := map[string]interface{}{
	}
	//判断请求参数
	if _, ok := request["skey"].(string); ok != true {
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"].(string))
		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	if _, ok := request["code"].(string); ok != true {
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"].(string))
		this.Data["json"] = response
		this.ServeJSON()
		return
	}

	//准备请求参数
	Skey := request["skey"].(string)
	code := utils.Decrypt(request["code"].(string))

	//配置缓存参数
	var redis_conf = map[string]string{
		"key":      utils.G_server_name,
		"conn":     utils.G_redis_addr + ":" + utils.G_redis_port,
		"dbNum":    utils.G_redis_dbnum,
		"password": utils.G_redis_password,
	}
	//将map进行转化成为json
	var redis_conf_js, _ = json.Marshal(redis_conf)

	//创建redis句柄
	bm, err := cache.NewCache("redis", string(redis_conf_js))
	if err != nil {
		logs.Info("redis连接失败", err)
		response["errno"] = utils.RECODE_DBERR
		response["errmsg"] = utils.RecodeText(response["errno"].(string))
		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	logs.Info("redis连接成功")

	//从缓存中取出哈希过的opid
	opid := bm.Get(Skey)
	if opid == nil {
		logs.Info("获取opid失败", err)
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"].(string))
		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	logs.Info("获取opid成功")
	//将取出的数据转换为string
	opidtemp, _ := redis.String(opid, nil)

	o := orm.NewOrm()
	var user models.User
	o.QueryTable("user").Filter("hxopid", opidtemp).One(&user)
	logs.Info("查询用户数据成功")

	//更新小关星数
	key := "mini" + code + user.Hxopid
	logs.Info(key)
	var minis [20]int
	ministemp_s := bm.Get(key)
	if ministemp_s == nil {
		logs.Info("redis中没有过关次数数据")
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"].(string))
		this.Data["json"] = response
		this.ServeJSON()
		return
	}

	//解码
	err = json.Unmarshal(ministemp_s.([]byte), &minis)
	if err != nil {
		logs.Info("解码失败", err)
	}
	logs.Info("获取小关次数成功")

	var cypminis []string

	for _, v := range minis {
		mini := utils.Encrypt(strconv.Itoa(v))
		cypminis = append(cypminis, mini)
	}
	response["grade"] = cypminis
	response["errno"] = utils.RECODE_OK
	response["errmsg"] = utils.RecodeText(response["errno"].(string))
	this.Data["json"] = response
	this.ServeJSON()
	return
}

//每日礼包
func (this *Controller) Package() {
	//打印日志
	logs.SetLogger(logs.AdapterFile, `{"filename":"./logs/api/aa.log"}`)
	logs.Info("Leaf 叶子变化服务 /api/connect/package")

	//接收请求参数
	var request map[string]interface{}
	if err := json.NewDecoder(this.Ctx.Request.Body).Decode(&request); err != nil {
		return
	}
	//定义反回数据
	response := map[string]string{
	}
	//判断请求参数
	if _, ok := request["skey"].(string); ok != true {
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"])

		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	if _, ok := request["code"].(string); ok != true {
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"])

		this.Data["json"] = response
		this.ServeJSON()
		return
	}

	//准备请求参数
	Skey := request["skey"].(string)
	Type := utils.Decrypt(request["code"].(string))

	//配置缓存参数
	var redis_conf = map[string]string{
		"key":      utils.G_server_name,
		"conn":     utils.G_redis_addr + ":" + utils.G_redis_port,
		"dbNum":    utils.G_redis_dbnum,
		"password": utils.G_redis_password,
	}
	//将map进行转化成为json
	var redis_conf_js, _ = json.Marshal(redis_conf)

	//创建redis句柄
	bm, err := cache.NewCache("redis", string(redis_conf_js))
	if err != nil {
		logs.Info("redis连接失败", err)
		response["errno"] = utils.RECODE_DBERR
		response["errmsg"] = utils.RecodeText(response["errno"])
		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	logs.Info("redis连接成功")
	//从缓存中取出哈希过的opid
	opid := bm.Get(Skey)
	if opid == nil {
		logs.Info("获取opid失败", err)
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"])
		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	logs.Info("获取opid成功")
	//将取出的数据转换为string
	opidtemp, _ := redis.String(opid, nil)

	o := orm.NewOrm()
	var user models.User
	err = o.QueryTable("user").Filter("hxopid", opidtemp).One(&user)
	if err != nil {
		logs.Info("查询用户数据失败")
	}
	logs.Info("查询数据成功")

	if err != nil {
		logs.Info("解码失败", err)
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"])
		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	fmt.Println("请求参数为", Type)

	key := "package" + user.Hxopid
	sginTemp := bm.Get(key)
	if sginTemp == nil {
		pack := "3"
		logs.Info("生成新用户签到数")
		sginJson, _ := json.Marshal(&pack)

		err = bm.Put(key, sginJson, time.Second*99999999)
		if err != nil {
			logs.Info("签到数放入redis失败", err)
		}
		logs.Info("签到数放入redis成功")
	}
	sginTemp1 := bm.Get(key)
	logs.Info("获取过签到数成功")
	var sginData string
	err = json.Unmarshal(sginTemp1.([]byte), &sginData)
	if err != nil {
		logs.Info("解码失败++++++++++++++++++++", err)
	}

	switch Type {
	case "0":
		fmt.Println("查询礼包数成功")
		fmt.Println("++++++++++++++++++", sginData)
		fmt.Println()
		fmt.Println()
		fmt.Println()
		fmt.Println()
		fmt.Println()
		fmt.Println()
		fmt.Println()
		fmt.Println()
		//返回数据
		response["number"] = utils.Encrypt(sginData)
		response["errno"] = utils.RECODE_OK
		response["errmsg"] = utils.RecodeText(response["errno"])
		this.Data["json"] = response
		this.ServeJSON()
		return

	case "1":
		bbb, _ := strconv.Atoi(sginData)
		aaa := strconv.Itoa(bbb - 1)

		sginJson, _ := json.Marshal(&aaa)

		err = bm.Put(key, sginJson, time.Second*99999999)
		if err != nil {
			logs.Info("签到数放入redis失败", err)
		}
		logs.Info("签到数放入redis成功")

		//返回数据
		response["number"] = utils.Encrypt(aaa)
		response["errno"] = utils.RECODE_OK
		response["errmsg"] = utils.RecodeText(response["errno"])
		this.Data["json"] = response
		this.ServeJSON()
		return
	default:
		logs.Info("参数不合法")
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"])
		this.Data["json"] = response
		this.ServeJSON()
		return
	}

	//返回数据
	//response["number"] = utils.Encrypt(sginData)
	response["errno"] = utils.RECODE_OK
	response["errmsg"] = utils.RecodeText(response["errno"])
	this.Data["json"] = response
	this.ServeJSON()
	return
}

//限时最高分数
func (this *Controller) MaxNumber() {
	//打印日志
	//logs.SetLogger(logs.AdapterFile, `{"filename":"./logs/connect.log"}`)
	logs.Info("Leaf 限时最高分数 /api/connect/maxnumber")

	//接收请求参数
	var request map[string]interface{}
	if err := json.NewDecoder(this.Ctx.Request.Body).Decode(&request); err != nil {
		return
	}
	//定义反回数据
	response := map[string]string{
	}
	//判断请求参数
	if _, ok := request["skey"].(string); ok != true {
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"])

		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	if _, ok := request["code"].(string); ok != true {
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"])

		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	if _, ok := request["number"].(string); ok != true {
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"])

		this.Data["json"] = response
		this.ServeJSON()
		return
	}

	//准备请求参数
	Skey := request["skey"].(string)
	Type := utils.Decrypt(request["code"].(string))
	Number, _ := strconv.Atoi(utils.Decrypt(request["number"].(string)))

	fmt.Println(Number)
	//配置缓存参数
	var redis_conf = map[string]string{
		"key":      utils.G_server_name,
		"conn":     utils.G_redis_addr + ":" + utils.G_redis_port,
		"dbNum":    utils.G_redis_dbnum,
		"password": utils.G_redis_password,
	}
	//将map进行转化成为json
	var redis_conf_js, _ = json.Marshal(redis_conf)

	//创建redis句柄
	bm, err := cache.NewCache("redis", string(redis_conf_js))
	if err != nil {
		logs.Info("redis连接失败", err)
		response["errno"] = utils.RECODE_DBERR
		response["errmsg"] = utils.RecodeText(response["errno"])
		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	logs.Info("redis连接成功")
	//从缓存中取出哈希过的opid
	opid := bm.Get(Skey)
	if opid == nil {
		logs.Info("获取opid失败", err)
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"])
		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	logs.Info("获取opid成功")
	//将取出的数据转换为string
	opidtemp, _ := redis.String(opid, nil)

	o := orm.NewOrm()
	var user models.User
	err = o.QueryTable("user").Filter("hxopid", opidtemp).One(&user)
	if err != nil {
		logs.Info("查询用户数据失败")
	}
	logs.Info("查询数据成功")

	switch Type {
	case "0":
		logs.Info("限时模式最高得分为", user.TimeLevel)

	case "1":
		if Number > user.TimeLevel {
			user.TimeLevel = Number
			o.Update(&user)
			logs.Info("记录限时模式最高得分为", Number)
		}

	default:
		logs.Info("参数不合法")
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"])
		this.Data["json"] = response
		this.ServeJSON()
		return
	}

	//返回数据
	response["number"] = utils.Encrypt(strconv.Itoa(user.TimeLevel))
	response["errno"] = utils.RECODE_OK
	response["errmsg"] = utils.RecodeText(response["errno"])
	this.Data["json"] = response
	this.ServeJSON()
	return
}

//开始过关
func (this *Controller) Start() {
	//打印日志
	//logs.SetLogger(logs.AdapterFile, `{"filename":"./logs/connect.log"}`)
	logs.Info("Leaf 叶子变化服务 /api/connect/start")

	//接收请求参数
	var request map[string]interface{}
	if err := json.NewDecoder(this.Ctx.Request.Body).Decode(&request); err != nil {
		return
	}
	//定义反回数据
	response := map[string]string{
	}
	//判断请求参数
	if _, ok := request["skey"].(string); ok != true {
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"])

		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	if _, ok := request["maxnumber"].(string); ok != true {
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"])

		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	if _, ok := request["mininumber"].(string); ok != true {
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"])

		this.Data["json"] = response
		this.ServeJSON()
		return
	}

	//准备请求参数
	sKey := request["skey"].(string)
	maxNumber := utils.Decrypt(request["maxnumber"].(string))
	miniNumber := utils.Decrypt(request["mininumber"].(string))

	//配置缓存参数
	var redis_conf = map[string]string{
		"key": utils.G_server_name,
		//127.0.0.1:6379
		"conn":     utils.G_redis_addr + ":" + utils.G_redis_port,
		"dbNum":    utils.G_redis_dbnum,
		"password": utils.G_redis_password,
	}
	//将map进行转化成为json
	var redis_conf_js, _ = json.Marshal(redis_conf)

	//创建redis句柄
	bm, err := cache.NewCache("redis", string(redis_conf_js))
	if err != nil {
		logs.Info("redis连接失败", err)
		response["errno"] = utils.RECODE_DBERR
		response["errmsg"] = utils.RecodeText(response["errno"])
		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	logs.Info("redis连接成功")
	//从缓存中取出哈希过的opid
	opid := bm.Get(sKey)
	if opid == nil {
		logs.Info("获取opid失败", err)
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"])
		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	logs.Info("获取opid成功")
	//将取出的数据转换为string
	opidtemp, _ := redis.String(opid, nil)
	logs.Info(opidtemp)

	number := ""
	timeNumber := ""

	o := orm.NewOrm()
	var maxs []models.MiniConfig
	_, err = o.QueryTable("mini_config").Filter("maxnumber", maxNumber).All(&maxs)
	if err != nil {
		logs.Info("查询小关数据失败")
	}
	logs.Info("查询小关数据成功")

	for _, v := range maxs {
		if v.MiniNumber == miniNumber {
			number = v.Number
			timeNumber = v.Time
		}
	}

	//返回数据
	response["number"] = utils.Encrypt(number)
	response["time"] = utils.Encrypt(timeNumber)
	response["errno"] = utils.RECODE_OK
	response["errmsg"] = utils.RecodeText(response["errno"])
	this.Data["json"] = response
	this.ServeJSON()
	return
}

//成功过关
func (this *Controller) Pass() {
	//打印日志
	//logs.SetLogger(logs.AdapterFile, `{"filename":"./logs/connect.log"}`)
	logs.Info("SetNowCheckpoint 成功过关 /api/connect/pass")

	//接收请求参数
	var request map[string]interface{}
	if err := json.NewDecoder(this.Ctx.Request.Body).Decode(&request); err != nil {
		return
	}
	//定义反回数据
	response := map[string]interface{}{
	}
	//判断请求参数
	if _, ok := request["skey"].(string); ok != true {
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"].(string))
		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	if _, ok := request["maxnumber"].(string); ok != true {
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"].(string))
		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	if _, ok := request["mininumber"].(string); ok != true {
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"].(string))
		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	if _, ok := request["Grade"].(string); ok != true {
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"].(string))
		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	//准备请求参数
	Skey := request["skey"].(string)
	maxNumber, _ := strconv.Atoi(utils.Decrypt(request["maxnumber"].(string)))
	miniNumber, _ := strconv.Atoi(utils.Decrypt(request["mininumber"].(string)))
	Grade, _ := strconv.Atoi(utils.Decrypt(request["Grade"].(string)))

	//配置缓存参数
	var redis_conf = map[string]string{
		"key":      utils.G_server_name,
		"conn":     utils.G_redis_addr + ":" + utils.G_redis_port,
		"dbNum":    utils.G_redis_dbnum,
		"password": utils.G_redis_password,
	}
	//将map进行转化成为json
	var redis_conf_js, _ = json.Marshal(redis_conf)

	//创建redis句柄
	bm, err := cache.NewCache("redis", string(redis_conf_js))
	if err != nil {
		logs.Info("redis连接失败", err)
		response["errno"] = utils.RECODE_DBERR
		response["errmsg"] = utils.RecodeText(response["errno"].(string))
		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	logs.Info("redis连接成功")

	//从缓存中取出哈希过的opid
	opid := bm.Get(Skey)
	if opid == nil {
		logs.Info("获取opid失败", err)
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"].(string))
		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	logs.Info("获取opid成功")
	//将取出的数据转换为string
	opidtemp, _ := redis.String(opid, nil)

	o := orm.NewOrm()
	var user models.User
	o.QueryTable("user").Filter("hxopid", opidtemp).One(&user)
	logs.Info("查询用户数据成功")

	//更新用户过关数据
	userLev := user.MaxLevel
	if miniNumber == 20 {

		if ((maxNumber-1)*20 + miniNumber) >= userLev {
			user.MaxConfig = strconv.Itoa(maxNumber + 1)
			user.MiniConfig = strconv.Itoa(1)
			user.MaxLevel = (maxNumber-1)*20 + miniNumber
		}

		o.Update(&user)
	} else {
		usermini, _ := strconv.Atoi(user.MiniConfig)
		if miniNumber >= usermini {
			user.MiniConfig = strconv.Itoa(miniNumber + 1)
		}
		userLev := user.MaxLevel
		if ((maxNumber-1)*20 + miniNumber) > userLev {
			user.MaxLevel = (maxNumber-1)*20 + miniNumber
		}
		o.Update(&user)
	}

	//更新小关星数
	key := "mini" + strconv.Itoa(maxNumber) + user.Hxopid
	ministemp_s := bm.Get(key)
	if ministemp_s == nil {
		logs.Info("当前用户无小关数据")
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"].(string))
		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	//解码

	var minis [20]int
	err = json.Unmarshal(ministemp_s.([]byte), &minis)
	if err != nil {
		logs.Info("解码失败", err)
	}
	logs.Info("获取小关次数成功")
	if Grade > minis[miniNumber-1] {
		minis[miniNumber-1] = Grade
	}
	//定义总星数
	gradeNumber := 0
	for _, v := range minis {
		gradeNumber += v
	}
	logs.Info("当前小关总星数为", gradeNumber)
	minisJson, _ := json.Marshal(&minis)
	err = bm.Put(key, minisJson, time.Second*99999999)
	if err != nil {
		logs.Info("小关星数放入redis失败", err)
	}
	logs.Info("小关星数放入redis成功")

	//更新大关数据
	key1 := "max" + user.Hxopid
	var maxs []models.MaxConfig

	numbertemp_s := bm.Get(key1)
	if numbertemp_s == nil {
		logs.Info("当前用户无大关数据")
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"].(string))
		this.Data["json"] = response
		this.ServeJSON()
		return
	}

	//解码
	err = json.Unmarshal(numbertemp_s.([]byte), &maxs)
	if err != nil {
		logs.Info("解码失败")
	}
	logs.Info("获取过关次数成功")
	state := "0"
	if miniNumber%20 == 0 {
		state = "1"
		maxs[maxNumber].Status = "1"
		user.MaxConfig = strconv.Itoa(maxNumber + 1)
		o.Update(&user)
	}
	//判断
	userMiniN, _ := strconv.Atoi(maxs[maxNumber-1].Usernumber)
	if miniNumber > userMiniN {
		maxs[maxNumber-1].Usernumber = strconv.Itoa(miniNumber)
	}
	maxs[maxNumber-1].Proportion = fmt.Sprintf("%.2f", float32(gradeNumber)/60)
	maxsjson, _ := json.Marshal(&maxs)
	err = bm.Put(key1, maxsjson, time.Second*99999999)
	if err != nil {
		logs.Info("过关数放入redis失败", err)
	}
	logs.Info("过关数放入redis成功")

	response["state"] = state
	response["errno"] = utils.RECODE_OK
	response["errmsg"] = utils.RecodeText(response["errno"].(string))
	this.Data["json"] = response
	this.ServeJSON()
	return
}

//钻石变化
func (this *Controller) Diamond() {
	//打印日志
	//logs.SetLogger(logs.AdapterFile, `{"filename":"./logs/connect.log"}`)
	logs.Info("Leaf 叶子变化服务 /api/connect/diamond")

	//接收请求参数
	var request map[string]interface{}
	if err := json.NewDecoder(this.Ctx.Request.Body).Decode(&request); err != nil {
		return
	}
	//定义反回数据
	response := map[string]string{
	}
	//判断请求参数
	if _, ok := request["skey"].(string); ok != true {
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"])
		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	if _, ok := request["code"].(string); ok != true {
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"])
		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	if _, ok := request["number"].(string); ok != true {
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"])
		this.Data["json"] = response
		this.ServeJSON()
		return
	}

	//准备请求参数
	Skey := request["skey"].(string)
	Type := utils.Decrypt(request["code"].(string))
	Number := utils.Decrypt(request["number"].(string))

	//配置缓存参数
	var redis_conf = map[string]string{
		"key": utils.G_server_name,
		//127.0.0.1:6379
		"conn":     utils.G_redis_addr + ":" + utils.G_redis_port,
		"dbNum":    utils.G_redis_dbnum,
		"password": utils.G_redis_password,
	}
	//将map进行转化成为json
	var redis_conf_js, _ = json.Marshal(redis_conf)

	//创建redis句柄
	bm, err := cache.NewCache("redis", string(redis_conf_js))
	if err != nil {
		logs.Info("redis连接失败", err)
		response["errno"] = utils.RECODE_DBERR
		response["errmsg"] = utils.RecodeText(response["errno"])
		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	logs.Info("redis连接成功")
	//从缓存中取出哈希过的opid
	opid := bm.Get(Skey)
	if opid == nil {
		logs.Info("获取opid失败", err)
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"])
		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	logs.Info("获取opid成功")
	//将取出的数据转换为string
	opidtemp, _ := redis.String(opid, nil)

	o := orm.NewOrm()
	var user models.User
	err = o.QueryTable("user").Filter("hxopid", opidtemp).One(&user)
	if err != nil {
		logs.Info("查询用户数据失败")
	}
	logs.Info("查询数据成功")

	usergold, _ := strconv.Atoi(user.UserMoney)
	num, _ := strconv.Atoi(Number)

	//判断参数
	if num < 0 {
		logs.Info("金币数不合法")
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"])
		this.Data["json"] = response
		this.ServeJSON()
		return
	}

	switch Type {
	case "1":
		user.UserMoney = strconv.Itoa(usergold + num)
		user.UserMaxMoney = strconv.Itoa(usergold + num)
		o.Update(&user)
		logs.Info("增加金币成功", num)

	case "2":
		if usergold < num {
			logs.Info("金币数不合法")
			response["errno"] = utils.RECODE_PARAMERR
			response["errmsg"] = utils.RecodeText(response["errno"])
			this.Data["json"] = response
			this.ServeJSON()
			return
		}
		user.UserMoney = strconv.Itoa(usergold - num)
		o.Update(&user)
		logs.Info("减少金币成功", num)

	case "0":
		logs.Info("查询叶子数成功")

	default:
		logs.Info("参数不合法")
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"])
		this.Data["json"] = response
		this.ServeJSON()
		return
	}

	//返回数据
	response["number"] = utils.Encrypt(user.UserMoney)
	response["maxnumber"] = utils.Encrypt(user.UserMaxMoney)
	response["errno"] = utils.RECODE_OK
	response["errmsg"] = utils.RecodeText(response["errno"])
	this.Data["json"] = response
	this.ServeJSON()
	return
}

//随机关卡
func (this *Controller) Random() {
	//打印日志
	//logs.SetLogger(logs.AdapterFile, `{"filename":"./logs/connect.log"}`)
	logs.Info("SetMaxCheckpoint 设置最高关卡服务 /api/connect/random")

	//接收请求参数
	var request map[string]interface{}
	if err := json.NewDecoder(this.Ctx.Request.Body).Decode(&request); err != nil {
		return
	}
	//定义反回数据
	response := map[string]interface{}{
	}
	//判断请求参数
	if _, ok := request["skey"].(string); ok != true {
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"].(string))

		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	//准备请求参数
	Skey := request["skey"].(string)

	//配置缓存参数
	var redis_conf = map[string]string{
		"key": utils.G_server_name,
		//127.0.0.1:6379
		"conn":     utils.G_redis_addr + ":" + utils.G_redis_port,
		"dbNum":    utils.G_redis_dbnum,
		"password": utils.G_redis_password,
	}
	//将map进行转化成为json
	var redis_conf_js, _ = json.Marshal(redis_conf)

	//创建redis句柄
	bm, err := cache.NewCache("redis", string(redis_conf_js))
	if err != nil {
		logs.Info("redis连接失败", err)
		response["errno"] = utils.RECODE_DBERR
		response["errmsg"] = utils.RecodeText(response["errno"].(string))
		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	logs.Info("redis连接成功")
	//从缓存中取出哈希过的opid
	opid := bm.Get(Skey)
	if opid == nil {
		logs.Info("获取opid失败", err)
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"].(string))
		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	logs.Info("获取opid成功")
	//将取出的数据转换为string
	opidtemp, _ := redis.String(opid, nil)
	logs.Info(opidtemp)
	o := orm.NewOrm()
	var Random []models.Random
	num, err := o.QueryTable("random").All(&Random)
	if err != nil {
		logs.Info("查询随机关卡数据失败")
	}
	logs.Info("查询随机关卡数据成功")
	logs.Info(num)

	var temp []models.Random
	for i := 0; i < 30; i++ {
		rand.Seed(time.Now().UnixNano())
		randa := rand.Intn(len(Random))
		fmt.Println(randa)
		temp = append(temp, Random[randa])
	}

	var tets []models.Random
	for _, v := range temp {
		var tet models.Random
		tet.Number =utils.Encrypt(v.Number)
		tet.MaxNumber =utils.Encrypt(v.MaxNumber)
		tet.Good =utils.Encrypt(v.Good)
		tet.Perfect = utils.Encrypt(v.Perfect)
		tets =append(tets,tet)
	}

	//返回数据
	response["checkpoint"] = tets
	response["errno"] = utils.RECODE_OK
	response["errmsg"] = utils.RecodeText(response["errno"].(string))
	this.Data["json"] = response
	//调用sercejson方法返回json数据
	this.ServeJSON()
	return
}

//红点数据
func (this *Controller) Remind() {
	//打印日志
	//logs.SetLogger(logs.AdapterFile, `{"filename":"./logs/connect.log"}`)
	logs.Info("SetMaxCheckpoint 设置最高关卡服务 /api/connect/remind")

	//接收请求参数
	var request map[string]interface{}
	if err := json.NewDecoder(this.Ctx.Request.Body).Decode(&request); err != nil {
		return
	}
	//定义反回数据
	response := map[string]string{
	}
	//判断请求参数
	if _, ok := request["skey"].(string); ok != true {
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"])

		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	//准备请求参数
	Skey := request["skey"].(string)

	//配置缓存参数
	var redis_conf = map[string]string{
		"key":      utils.G_server_name,
		"conn":     utils.G_redis_addr + ":" + utils.G_redis_port,
		"dbNum":    utils.G_redis_dbnum,
		"password": utils.G_redis_password,
	}
	//将map进行转化成为json
	var redis_conf_js, _ = json.Marshal(redis_conf)

	//创建redis句柄
	bm, err := cache.NewCache("redis", string(redis_conf_js))
	if err != nil {
		logs.Info("redis连接失败", err)
		response["errno"] = utils.RECODE_DBERR
		response["errmsg"] = utils.RecodeText(response["errno"])
		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	logs.Info("redis连接成功")
	//从缓存中取出哈希过的opid
	opid := bm.Get(Skey)
	if opid == nil {
		logs.Info("获取opid失败", err)
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"])
		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	logs.Info("获取opid成功")
	//将取出的数据转换为string
	opidtemp, _ := redis.String(opid, nil)

	o := orm.NewOrm()
	var user models.User
	err = o.QueryTable("user").Filter("hxopid", opidtemp).One(&user)
	if err != nil {
		logs.Info("查询用户数据失败")
	}
	logs.Info("查询数据成功")

	PackageNumber := "0"
	tempPack, _ := strconv.Atoi(user.Package)

	if tempPack == 0 {
		PackageNumber = "1"
	}

	//返回数据
	response["sign"] = user.SignNumber
	response["package"] = PackageNumber
	response["errno"] = utils.RECODE_OK
	response["errmsg"] = utils.RecodeText(response["errno"])
	this.Data["json"] = response
	//调用sercejson方法返回json数据
	this.ServeJSON()
	return
}

//签到
func (this *Controller) Sign() {
	logs.Info("Sign 签到 /api/connect/sign")

	//接收请求参数
	var request map[string]interface{}
	if err := json.NewDecoder(this.Ctx.Request.Body).Decode(&request); err != nil {
		return
	}
	//定义反回数据
	response := map[string]interface{}{
	}
	//判断请求参数
	if _, ok := request["skey"].(string); ok != true {
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"].(string))

		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	if _, ok := request["code"].(string); ok != true {
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"].(string))

		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	//准备请求参数
	Skey := request["skey"].(string)

	code := utils.Decrypt(request["code"].(string))

	//配置缓存参数
	var redis_conf = map[string]string{
		"key":      utils.G_server_name,
		"conn":     utils.G_redis_addr + ":" + utils.G_redis_port,
		"dbNum":    utils.G_redis_dbnum,
		"password": utils.G_redis_password,
	}
	//将map进行转化成为json
	var redis_conf_js, _ = json.Marshal(redis_conf)

	//创建redis句柄
	bm, err := cache.NewCache("redis", string(redis_conf_js))
	if err != nil {
		logs.Info("redis连接失败", err)
		response["errno"] = utils.RECODE_DBERR
		response["errmsg"] = utils.RecodeText(response["errno"].(string))
		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	logs.Info("redis连接成功")
	//从缓存中取出哈希过的opid
	opid := bm.Get(Skey)
	if opid == nil {
		logs.Info("获取opid失败", err)
		response["errno"] = utils.RECODE_PARAMERR
		response["errmsg"] = utils.RecodeText(response["errno"].(string))
		this.Data["json"] = response
		this.ServeJSON()
		return
	}
	logs.Info("获取opid成功")
	//将取出的数据转换为string
	opidtemp, _ := redis.String(opid, nil)

	o := orm.NewOrm()
	var user models.User
	err = o.QueryTable("user").Filter("hxopid", opidtemp).One(&user)
	if err != nil {
		logs.Info("查询用户数据失败")
	}
	logs.Info("查询数据成功")

	key := "sgin" + user.Hxopid
	sginTemp := bm.Get(key)
	if sginTemp == nil {
		sgin := [...]string{"1", "2", "2", "2", "2", "2", "2"}
		logs.Info("生成新用户签到数")
		sginJson, _ := json.Marshal(&sgin)

		err = bm.Put(key, sginJson, time.Second*99999999)
		if err != nil {
			logs.Info("签到数放入redis失败", err)
		}
		logs.Info("签到数放入redis成功")
	}
	sginTemp1 := bm.Get(key)
	logs.Info("获取过签到数成功")
	var sginData []string
	err = json.Unmarshal(sginTemp1.([]byte), &sginData)
	if err != nil {
		logs.Info("解码失败")
	}
	switch code {
	case "0":
		for i := 0; i < len(sginData); i++ {
			sginData[i] = utils.Encrypt(sginData[i])
		}
		//返回数据
		response["sign"] = sginData
		response["errno"] = utils.RECODE_OK
		response["errmsg"] = utils.RecodeText(response["errno"].(string))
		this.Data["json"] = response
		this.ServeJSON()
		return
	case "1":
		beego.Info(sginData)
		for k, v := range sginData {
			if v == "1" {
				signNumber, _ := strconv.Atoi(user.SignNumber)

				user.SignNumber = strconv.Itoa(signNumber + 1)

				_, err = o.Update(&user)
				if err != nil {
					logs.Info("插入最高关卡数失败", err)
				}

				sginData[k] = "0"
				//sginData[k+1] = "1"
				sginJson1, _ := json.Marshal(&sginData)
				err = bm.Put(key, sginJson1, time.Second*99999999)
				if err != nil {
					logs.Info("签到数放入redis失败", err)
				}
				logs.Info("签到数放入redis成功")
				break
			}
		}
	}
	//返回数据
	response["errno"] = utils.RECODE_OK
	response["errmsg"] = utils.RecodeText(response["errno"].(string))
	this.Data["json"] = response
	this.ServeJSON()
	return
}
