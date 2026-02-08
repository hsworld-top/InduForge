// publisher.js

const mqtt = require("mqtt");
const dayjs = require("dayjs");
const { TIME_FORMAT } = require("../src/constants/time");

// 1. MQTT 服务器连接选项
const options = {
  host: "localhost",
  port: 1883,
  // 如果需要认证，可以添加用户名和密码
  username: 'admin',
  password: 'public',
  clientId: `mqtt_publisher_${Math.random().toString(16).substr(2, 8)}`,
};

// 2. 连接到 MQTT 服务器
const client = mqtt.connect(options);

// 3. 定义要发布的主题
const topic = "test";

// 连接成功后的回调
client.on("connect", () => {
  console.log("成功连接到 MQTT 服务器");

  // 4. 设置定时器，每隔 2 秒发布一次数据
  setInterval(() => {
    // 5. 生成模拟数据
    const payload = {
      N: "test",
      V: Math.floor(Math.random() * 100) + 1,
      Q: 1,
      T: dayjs().format(TIME_FORMAT),
    };

    // 6. 将数据转换为字符串并发布
    client.publish(topic, JSON.stringify(payload), (err) => {
      if (err) {
        console.error("发布消息失败:", err);
      } else {
        console.log(`已发布消息到主题 ${topic}:`, payload);
      }
    });
  }, 2000); // 每 2000 毫秒（2秒）执行一次
});

// 连接错误的回调
client.on("error", (err) => {
  console.error("连接错误:", err);
  client.end(); // 结束连接
});

// 关闭连接的回调
client.on("close", () => {
  console.log("连接已关闭");
});
