# 部署说明

生产环境不要把真实 Binance 密钥写入仓库或配置文件。

推荐做法：

1. 把二进制部署到服务器，例如 `/opt/grid-trade/trader`
2. 把配置文件部署到 `/opt/grid-trade/configs/example.yaml`
3. 把真实环境变量写入仅服务器可读的 env 文件，例如 `/etc/grid-trade/grid-trade.env`
4. 用 `systemd` 读取这个 env 文件并启动服务

示例 env 文件内容：

```bash
BINANCE_API_KEY=your-real-key
BINANCE_SECRET_KEY=your-real-secret
TRADER_CONFIG=/opt/grid-trade/configs/example.yaml
TRADER_HTTP_ADDR=:18080
```

安全要求：

- env 文件权限设为 `600`
- 所属用户设为运行服务的系统用户
- 不要把真实 env 文件提交到 git
- 调试控制台只显示“是否缺少环境变量”和错误信息，不显示密钥值

如果使用 `systemd`，可参考同目录下的 `grid-trade.service`。
