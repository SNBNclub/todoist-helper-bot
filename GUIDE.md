


/var/log/nginx/nginx_error.log




**## Resources**

### Philosophy and code style
- [How to write Go code](https://go.dev/doc/code#Organization)
- [Google code style](https://google.github.io/styleguide/go/decisions#variable-names)

### Structure
- [Organizing go module](https://go.dev/doc/modules/layout)
- [Standart project layout](https://github.com/golang-standards/project-layout)
- [About standart layout](https://medium.com/evendyne/getting-started-with-go-project-structure-ab8814ded9c3)

### Creating bots
- [Telegram bot API](https://core.telegram.org/bots/api#callbackquery)
- with [tgbotapi](https://medium.com/@nbenliogludev/how-to-build-a-to-do-list-telegram-bot-with-the-golang-postgresql-database-b77b1ec014ba)

### Logging
- [Custom zap loggers](https://betterstack.com/community/guides/logging/go/zap/)

### DataBases
- [ORM with Migrations tool](https://gorm.io/docs/)

### Setting up vnc
- [Console](https://coddswallop.wordpress.com/2012/05/09/ubuntu-12-04-precise-pangolin-complete-vnc-server-setup/)

### JSON to Go models
- [First](https://mholt.github.io/json-to-go/)

## Setting up own server
- Get server
  1. Get sever with public IP. Check ip - `curl ifconfig.me`
  2. Buy domain name, add DNS recording with `A` type (which means redirect from public IP). 
  3. Domain will refresh on servers in 24 hours (DNS propagation). You can check it with `ping example.com`.
  4. Profit!!
- On local computer
  1. Fix the computer IP address in your router settings.
  2. Get public IP from your internet provider.
  3. Pass inner ports to global network in router settings.


## This project

### Tasks:
- [ ] Timer 
  - [ ] where to store current time
    LLM request "This will be time tracker with start stop funcitons for the tasks and with end. when user ends task - we store result. is it correct to store intermediate results and current time on user device? or how to do this timer if we don't ha"
- [ ] where it is better to init packeges? should we have main as connector and all dependencies specify there, or we can just have `func init()` in each package and import them
- [ ] how to use configs/config and import it if i prefer this way to loaddotenv
- [ ] writing guide for structure
  - [ ] project layout
  - [ ] migrations & sql queries
    - [ ] auto generate sql querise, auto generate models
    - [ ] sql constraints??
- [ ] simple integration with labels
  - [ ] label to set time for the task
    - [ ] we need format "%HH%MM"
  - [ ] label to ask, how much we spent when task ends
  - [ ] tests immitating todoist webhook 
- [ ] add all Grok resources

### Structure 
my-telegram-bot/
├── /cmd/
│   └── /bot/
│       └── main.go        
├── /internal/
│   ├── /bot/
│   │   ├── /handlers/        
│   │   ├── /commands/        
│   │   └── bot.go            
│   ├── /service/
│   │   └── todoist.go        
│   ├── /repository/          
│   └── /models/              
├── docker-compose.yml        
├── Dockerfile              
├── go.mod
└── go.sum