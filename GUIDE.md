# Resources

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

### Reflect example
- [Strcut validation](https://medium.com/@anajankow/fast-check-if-all-struct-fields-are-set-in-golang-bba1917213d2)

## Setting up own server
  1. Get sever with public IP. Check ip - `curl ifconfig.me`
  2. Buy domain name, add DNS recording with `A` type (which means redirect from public IP). 
  3. Domain will refresh on servers in 24 hours (DNS propagation). You can check it with `ping example.com`.
  4. Generating SSL certificates with Let's Encrypt.
     1. `sudo apt install certbot -y`
     2. `sudo certbot certonly --standalone -d sbn.online -d www.sbn.online`
     3. Certs will be at - `/etc/letsencrypt/live/yourname.ddns.net/`.
     4. Certboot has autorenewal but you can do it manually `sudo certbot renew --dry-run`.
     5. Set up ngnix
        1. In `/etc/nginx/sites-available/myapp.conf` create config file like
        ```nginx
        server {
            listen 443 ssl;  # Enable SSL
            server_name myapp.com;

            # Paths to your Let's Encrypt certificates
            ssl_certificate /etc/letsencrypt/live/myapp.com/fullchain.pem;
            ssl_certificate_key /etc/letsencrypt/live/myapp.com/privkey.pem;

            location / {
                proxy_pass http://localhost:8080;  # Backend uses HTTP
                proxy_set_header Host $host;
                proxy_set_header X-Real-IP $remote_addr;
            }
        }

        # Redirect HTTP to HTTPS
        server {
            listen 80;
            server_name myapp.com;
            return 301 https://$host$request_uri;
        }
        ```
        2. Activate it (create symbolic link) `sudo ln -s /etc/nginx/sites-available/myapp.conf /etc/nginx/sites-enabled/`
        3. Test syntax `sudo nginx -t`
        1. `sudo systemctl restart nginx`
