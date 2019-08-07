package routers

import (
	"connect/controllers"
	"github.com/astaxie/beego"
)

func init() {

	beego.Router("/api/connect/ping", &controllers.Controller{}, "head:Ping")

	//登陆服务
	beego.Router("/api/connect/login", &controllers.Controller{}, "post:Login")

	//获取邀请新人列表
	beego.Router("/api/connect/pullnew", &controllers.Controller{}, "post:PullNew")

	//服务器排名服务
	beego.Router("/api/connect/ranking", &controllers.Controller{}, "post:Ranking")

	//限时模式世界排名
	beego.Router("/api/connect/timeranking", &controllers.Controller{}, "post:TimeRanking")

	//服务器时间服务
	beego.Router("/api/connect/nowtime", &controllers.Controller{}, "post:NowTime")

	//获取用户过关数据
	beego.Router("/api/connect/getcheckpoint", &controllers.Controller{}, "post:GetCheckpoint")

	//获取大关数据
	beego.Router("/api/connect/maxconfig", &controllers.Controller{}, "post:MaxConfig")

	//获取小关数据
	beego.Router("/api/connect/miniconfig", &controllers.Controller{}, "post:MiniConfig")

	//每日礼包
	beego.Router("/api/connect/package", &controllers.Controller{}, "post:Package")

	//限时最高分数
	beego.Router("/api/connect/maxnumber", &controllers.Controller{}, "post:MaxNumber")

	//开始过关
	beego.Router("/api/connect/start", &controllers.Controller{}, "post:Start")

	//成功过关
	beego.Router("/api/connect/pass", &controllers.Controller{}, "post:Pass")

	//钻石变化
	beego.Router("/api/connect/diamond", &controllers.Controller{}, "post:Diamond")

	//随机关卡
	beego.Router("/api/connect/random", &controllers.Controller{}, "post:Random")

	//红点数据
	beego.Router("/api/connect/remind", &controllers.Controller{}, "post:Remind")

	//记录签到
	beego.Router("/api/connect/sign", &controllers.Controller{}, "post:Sign")

}
