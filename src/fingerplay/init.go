package main

func Init() (err error) {
	DefaultAccountManager = NewAccountManager(Conf.EndpointDescribeUser, Conf.EndpointTransfer, Conf.EndpointLoginAI)
	DefaultLogicImpl = NewLogicImpl(DefaultAccountManager, Conf.Levels, Conf.OperateTimeoutSecond, Conf.MatchWaitSecond)
	DefaultRobotManager = NewRobotManager(Conf.RobotUid, Conf.RobotLifetimeSecond)

	mongoConfig := MongoConfig{}
	mongoConfig.serverAddr = Conf.MongoServerAddrs

	DefaultContext = NewContext(NewMongoManager(mongoConfig))

	DefaultStatisticsManager = NewStatisticsManager(DefaultContext)

	DefaultRiskController = NewRiskController(DefaultContext)

	return InitHttp(Conf.HttpBindAddr)
}
