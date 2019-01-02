# 搬砖小蚂蚁
    Mixin ID: 7000101695
    一个在Ocean ONE和ExinOne上寻找搬砖机会的机器人，若存在约1%的价格差，则开启自动交易，
    或Ocean ONE上买，ExinOne上卖，或Ocean ONE上卖，ExinOne上买。若有用户订阅，则会
    推送相应的套利机会，并给出交易的直达链接。大部分时间行情相当无聊，于是在小蚂蚁里链接一个聊天机器人。

## 运行
   进入demo目录，go build -o ant 编译，然后输入 ./ant run --ocean --exin 运行即可,需提供mysql和redis环境支持
