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
TELEGRAM_STATS_GROUP=<chat-id>
EOL

docker run -d --env-file env.list dinup24/vax-notifier
```
