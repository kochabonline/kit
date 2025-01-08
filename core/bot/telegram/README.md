# 快速开始

[电报机器人API文档](https://core.telegram.org/bots/api)


# 获取个人ID

- 使用`@userinfobot`获取个人id

- 给自己创建的机器人发送消息, 然后打开`https://api.telegram.org/bot(机器人TOKEN)/getUpdates`, 查找自己发的消息id字段


# 获取群组ID

需要将机器人拉取群组, 然后发送`/xx @机器人`, 最后打开`https://api.telegram.org/bot(机器人TOKEN)/getUpdates`, 获取发送消息的id字段(群组id以`-`开头)
