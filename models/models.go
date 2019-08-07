package models

import (
	//go操作数据库的模块
	_ "github.com/go-sql-driver/mysql"
	"connect/utils"
	"github.com/astaxie/beego/orm"
)

//用户
type User struct {
	Id                int           `json:"user_id"`                           //用户编号
	OpenId            string        `orm:"size(128)" json:"open_id"`           //从微信获取的用户Id
	UnionId           string        `orm:"size(128)" json:"union_id"`          //从微信获取的全平台唯一Id
	Hxopid            string        `orm:"size(256)" json:"hxopid"`            //哈希处理后的opid
	Name              string        `orm:"size(128)"  json:"name"`             //用户昵称
	Avatar_url        string        `orm:"size(256)" json:"avatar_url"`        //用户头像路径
	Gender            string        `orm:"size(128)" json:"gender"`            //性别
	Age               string        `orm:"size(128)" json:"age"`               //年龄
	Session_key       string        `orm:"size(128)" json:"session_key"`       //从微信获取的sessoing_key
	Skey              string        `orm:"size(128)" json:"s_key"`             //自定义登陆态
	Landing_time      string        `orm:"size(256)" json:"landing_time"`      //最后登陆时间
	Registration_time string        `orm:"size(256)" json:"registration_time"` //用户注册时间
	MaxLevel          int           `json:"max_level"`                         //最高关卡数,大关数*20+小关数
	MaxConfig         string        `orm:"size(256)" json:"max_config"`        //大关数
	MiniConfig        string        `orm:"size(256)" json:"mini_config"`       //小关数
	TimeLevel         int           `json:"time_level"`                        //限时模式关卡数
	UserMoney         string        `orm:"size(256)" json:"user_money"`        //当前钻石数
	UserMaxMoney      string        `orm:"size(256)" json:"user_money"`        //累计钻石数
	LevelTime         string        `orm:"size(256)" json:"level_time"`        //过关时间
	SignNumber        string        `orm:"size(256)" json:"sign_number"`       //登陆奖励
	Package           string        `orm:"size(256)" json:"package"`           //礼包
	Invitations       []*Invitation `orm:"reverse(many)" json:"invitations"`   //邀请的新人信息
}

//邀请新人的信息
type Invitation struct {
	Id                    int    `json:"Invitation_id"`
	User                  *User  `orm:"rel(fk)" json:"open_id"`                 //邀请人opid
	Invitation_OpenId     string `orm:"size(128)" json:"invitation_open_id"`    //从微信获取的用户Id
	Invitation_Avatar_url string `orm:"size(256)" json:"invitation_avatar_url"` //从微信获取的头像地址
	Invitation_Name       string `orm:"size(128)" json:"invitation_name"`       //用户昵称
	Invitation_time       string `orm:"size(128)" json:"invitation_time"`       //分享时间
	GetStatus             string `orm:"size(128)" json:"get_status"`            //奖励领取
}

//大关数据
type MaxConfig struct {
	Id         int    `json:"MaxConfig_id"`
	Code       string `orm:"size(128)" json:"code"`       //关卡id
	Name       string `orm:"size(128)" json:"name"`       //关卡名称
	Url        string `orm:"size(128)" json:"url"`        //美术资源url
	Bjurl      string `orm:"size(128)" json:"bjurl"`      //背景url
	Mininumber string `orm:"size(128)" json:"mininumber"` //包含小关数
	Proportion string `orm:"size(128)" json:"proportion"` //星数百分比
	Status     string `orm:"size(128)" json:"status"`     //解锁状态
	Usernumber string `orm:"size(128)" json:"usernumber"` //用户小关数
}

//过关条件
type MiniConfig struct {
	Id         int    `json:"MiniConfig_id"`
	MaxNumber  string `orm:"size(128)" json:"max_number"`  //大关数
	MiniNumber string `orm:"size(128)" json:"mini_number"` //小关数
	Number     string `orm:"size(128)" json:"number"`      //步数
	Time       string `orm:"size(128)" json:"time"`        //妙数
}

//随机关卡
type Random struct {
	Id        int    `json:"random_id"`
	Number    string `orm:"size(128)" json:"number"`    //随机小关数
	MaxNumber string `orm:"size(128)" json:"maxnumber"` //随机大关数
	Perfect   string `orm:"size(128)" json:"perfect"`   //perfect秒数
	Good      string `orm:"size(128)" json:"good"`      //good秒数
}

func init() {
	//注册mysql的驱动
	orm.RegisterDriver("mysql", orm.DRMySQL)
	// 设置默认数据库
	orm.RegisterDataBase("default", "mysql", utils.G_mysql_user+":"+utils.G_mysql_password+"@tcp("+utils.G_mysql_addr+":"+utils.G_mysql_port+")/"+utils.G_mysql_dbname+"?charset=utf8", 30)
	//orm.RegisterDataBase("default", "mysql", "root:1@tcp("+utils.G_mysql_addr+":"+utils.G_mysql_port+")/plant?charset=utf8", 30)

	//注册model
	orm.RegisterModel(new(User), new(Invitation), new(MaxConfig), new(Random), new(MiniConfig))

	// create table
	//第一个是别名
	//第二个是是否强制替换模块   如果表变更就将false 换成true 之后再换回来表就便更好来了
	//第三个参数是如果没有则同步或创建
	orm.RunSyncdb("default", false, true)
}
