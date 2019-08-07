package main

import (
	"encoding/json"
	"github.com/astaxie/beego/cache"
	_ "github.com/astaxie/beego/cache/redis"
	_ "github.com/gomodule/redigo/redis"
	_ "connect/routers"
	"github.com/astaxie/beego"
	"github.com/robfig/cron"
	"github.com/astaxie/beego/orm"
	"github.com/astaxie/beego/logs"
	"connect/models"
	"connect/utils"
	"time"
)

func main() {
	logs.SetLogger(logs.AdapterFile, `{"filename":"./logs/connect.log"}`)
	//logs.SetLogger("console", "")
	logs.EnableFuncCallDepth(true)
	logs.SetLogFuncCallDepth(3)

	//定时任务
	c := cron.New()
	c.AddFunc("0 0 0 * * ?", func() {
		//c.AddFunc("*/10 * * * * ?", func() {
		o := orm.NewOrm()
		var users []models.User
		o.QueryTable("user").All(&users)
		logs.Info("查询用户数据成功")

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
		}
		logs.Info("redis连接成功")

		//签到红点
		for i := 0; i < len(users); i++ {
			users[i].SignNumber = "0"
			o.Update(&users[i])
			logs.Info("签到红点成功")
		}

		//重置签到
		for i := 0; i < len(users); i++ {
			signkey := "sgin" + users[i].Hxopid
			logs.Info(signkey)
			sginTemp := bm.Get(signkey)
			if sginTemp == nil {
				logs.Info(users[i].Name,"签到数据为空")
				continue
			}
			logs.Info("获取过签到数成功")
			var sginData []string
			err = json.Unmarshal(sginTemp.([]byte), &sginData)
			if err != nil {
				logs.Info("解码失败")
			}

			for k, v := range sginData {
				if v == "1" {
					break
				}
				if v == "0" && k == 6 {
					beego.Info("领取七天，清楚数据成功")
					err := bm.Delete(signkey)
					if err != nil {
						logs.Info("删除签到数据失败")
					}
					break
				}
				if v == "2" {
					sginData[k] = "1"
					logs.Info("重置签到数据成功")
					sginJson, _ := json.Marshal(&sginData)
					err = bm.Put(signkey, sginJson, time.Second*99999999)
					if err != nil {
						logs.Info("签到数放入redis失败", err)
					}
					logs.Info("签到数放入redis成功")
					logs.Info(users[i].Name,"重置后数据为", sginData)
					break
				}
			}
		}

		//重置礼包
		for i := 0; i < len(users); i++ {
			signkey := "package" + users[i].Hxopid
			logs.Info(signkey)
			sginTemp := bm.Get(signkey)
			if sginTemp == nil {
				logs.Info(users[i].Name,"礼包数据为空")
				continue
			}
			logs.Info("获取过礼包数成功")
			var packageData string
			err = json.Unmarshal(sginTemp.([]byte), &packageData)
			if err != nil {
				logs.Info("解码失败")
			}

			packageData = "3"

			sginJson, _ := json.Marshal(&packageData)
			err = bm.Put(signkey, sginJson, time.Second*99999999)
			if err != nil {
				logs.Info("签到数放入redis失败", err)
			}
			logs.Info("签到数放入redis成功")

			logs.Info("礼包数重置成功")
		}

	})
	c.Start()

	beego.Run()
}
