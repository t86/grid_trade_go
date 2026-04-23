# 部署说明

生产环境不要把真实 Binance 密钥写入仓库、YAML 配置或公开环境变量模板。

推荐做法：

1. 把二进制部署到服务器，例如 `/opt/grid-trade/trader`
2. 把配置文件部署到 `/opt/grid-trade/configs/example.yaml`
3. 在服务器受控目录准备账号密钥文件，例如 `/etc/grid-trade/accounts`
4. 只把运行参数写入 env 文件，例如 `/etc/grid-trade/grid-trade.env`
5. 用 `systemd` 读取 env 文件并启动服务

示例配置片段：

```yaml
system:
  secret_dir: "/etc/grid-trade/accounts"
accounts:
  - name: primary
    enabled: true
    market_types: ["spot", "futures_um"]
    secret_ref: primary
```

示例 secret 文件 `/etc/grid-trade/accounts/primary.json`：

```json
{
  "api_key": "your-real-key",
  "secret_key": "your-real-secret"
}
```

示例 env 文件内容：

```bash
TRADER_CONFIG=/opt/grid-trade/configs/example.yaml
TRADER_HTTP_ADDR=:18080
```

安全要求：

- secret 文件目录建议权限 `700`
- secret JSON 文件权限设为 `600`
- secret 文件和 env 文件都只允许服务运行用户读取
- env 文件权限设为 `600`
- 所属用户设为运行服务的系统用户
- 不要把真实 secret 文件和 env 文件提交到 git
- 调试控制台只显示密钥引用、加载状态和错误信息，不显示密钥值

如果使用 `systemd`，可参考同目录下的 `grid-trade.service`。
