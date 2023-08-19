this example is to send a prompt and download the generated images to a data
directory

# HOW TO SET UP

### start server

```sh
cd server/
go run main.go
```

### set up ngrok

```sh
ngrok http 8080
```

### export ngrok url and replicate api token

```sh
# get ngrok url and export ngrok url.
NGROK_URL=$(curl -s localhost:4040/api/tunnels|jq -r ".tunnels[0].public_url")
export REPLICATE_NGROK_URL=$NGROK_URL

# https://replicate.com/account/api-tokens
export REPLICATE_API_TOKEN="YOUR_API_TOKEN"
```

### request image generation to replicate api

```sh
cd client
go run main.go
```
