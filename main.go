package main

import (
	"github.com/Idiotmann/cart/domain/repository"
	service2 "github.com/Idiotmann/cart/domain/service"
	"github.com/Idiotmann/cart/handler"
	cart "github.com/Idiotmann/cart/proto"
	"github.com/Idiotmann/common"
	"github.com/go-micro/plugins/v4/registry/consul"
	"github.com/go-micro/plugins/v4/wrapper/ratelimiter/uber"
	opentracing2 "github.com/go-micro/plugins/v4/wrapper/trace/opentracing"
	"github.com/jinzhu/gorm"
	_ "github.com/jinzhu/gorm/dialects/mysql"
	"github.com/opentracing/opentracing-go"
	"go-micro.dev/v4"
	"go-micro.dev/v4/registry"
	"log"
)

var QPS = 100

func main() {
	//配置中心
	consulConfig, err := common.GetConsulConfig("127.0.0.1", 8500, "micro/config")
	if err != nil {
		log.Fatal(err)
	}
	//注册中心
	consulReg := consul.NewRegistry(func(options *registry.Options) {
		options.Addrs = []string{"127.0.0.1:8500"}
	})
	//链路追踪
	t, io, err := common.NewTracer("go.micro.service.cart", "localhost:6831")
	if err != nil {
		log.Fatal(err)
	}
	defer io.Close()
	opentracing.SetGlobalTracer(t)

	//获取mysql配置,路径中不带前缀
	//mysql需要手动加载数据库驱动
	mysqlConfig, err := common.GetMysqlFromConsul(consulConfig, "mysql")
	if err != nil {
		log.Fatal(err)
	}
	// 数据库类型：mysql，数据库用户名：root，密码：Kk1503060325，数据库名字：micro
	db, err := gorm.Open("mysql", mysqlConfig.User+":"+mysqlConfig.Password+"@/"+mysqlConfig.Database+"?charset=utf8&parseTime=True&loc=Local")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	db.SingularTable(true) //禁用表名复数

	//创建表之后，就注释掉
	//repository.NewCartRepository(db).InitTable()

	service := micro.NewService(
		micro.Name("go.micro.service.cart"),
		micro.Version("latest"),
		micro.Address("127.0.0.1:8087"), //服务启动的地址
		micro.Registry(consulReg),       //注册中心
		//绑定链路追踪  服务端绑定handler,客户端绑定Client
		micro.WrapHandler(opentracing2.NewHandlerWrapper(opentracing.GlobalTracer())),
		//限流
		micro.WrapHandler(ratelimit.NewHandlerWrapper(QPS)), //每秒QPS个请求
	)
	//获取mysql配置,路径中不带前缀
	//mysql需要手动加载数据库驱动

	// Initialise service
	service.Init()
	cartDataService := service2.NewCartDataService(repository.NewCartRepository(db))

	// Register Handler
	cart.RegisterCartHandler(service.Server(), &handler.Cart{CartDataService: cartDataService})
	if err := service.Run(); err != nil {
		log.Fatal(err)
	}
}
