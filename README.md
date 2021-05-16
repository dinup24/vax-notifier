# COVID Vaccine Notifier

### Development environment
```
git clone https://github.com/dinup24/vax-notifier.git

cd vax-notifier
```
Build
```
go build .
```
Execute
```
go run .
```

### Build
```
docker build -t dinup24/vax-notifier .
```

### Deploy
```
cat > env.list <<EOL
TELEGRAM_TOKEN=<token>
CONFIG_FILE=config.yaml
STATS_TELEGRAM_GROUP=<chat-id>
POLLING_INTERVAL=60s
EOL

docker run -d --env-file env.list dinup24/vax-notifier
```

Deploy with a custom config file
```
cities:
  - name: Bangalore
    districtId:
    - 265
    - 276
    - 294
    channels:
    - minAge:
      - 18
      channelNname: vax-notifier [u45 - BLR]
      chatId: "@vaxu45blr"
  - name: Ernakulam
    districtId:
    - 307
    channels:
    - minAge:
      - 18
      channelNname: vax-notifier [u45 - EKM]
      chatId: "@vaxu45ekm"
```
