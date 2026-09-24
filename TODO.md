# TODO list for fiber backend template

## SOON:

0. update config (oauth can be disabled but fields are required. do it like the mailer service)

```
type GoogleOAuthConfig struct {
	Enabled      bool   `env:"GOOGLE_OAUTH_ENABLED" envDefault:"false"`
	ClientID     string `env:"GOOGLE_CLIENT_ID,required"`
	ClientSecret string `env:"GOOGLE_CLIENT_SECRET,required"`
	RedirectURL  string `env:"GOOGLE_REDIRECT_URL,required"`
}
```

1. create templates and implement mailer to auth service (email verification)
2. implement reset auth password (email verification!!)
3. users --C--RUD for admin role

## NOT SOON

1. make automatic install better and simplier

## ????

1. payments (cryptobot, yookassa) implement (will this be a template? should i move this into a separate template / branch?)
2. cover with tests
