---
sidebar_label: "Setup Fake Log Generation"
sidebar_position: 2
---
# Set up a “fake log creating” job (Optional)

Note: these commands are already there in the terraform scripts and this part is for explanation only and is a manual process compared to the automated script

Note you can use your logs

```bash
git clone https://github.com/js-ts/logrotate
cd logrotate
chmod +x ./log-rotate.sh
sudo ./log-rotate.sh
```

To rotate log files every hour on Ubuntu, you can use the logrotate utility. Follow these steps to configure hourly log rotation:

Install logrotate if it’s not already installed:

```bash
sudo apt-get updatesudo apt-get install logrotate
```

1. Create a new configuration file for your log file. For example, if your log file is located at **`/var/log/myapp.log`**, create a new file called **`/etc/logrotate.d/myapp`** with the following content:

```bash
sudo nano /etc/logrotate.d/myapp
```

Then, add the following configuration:

```bash
/home/<your-username>/logrotate/logs/fake_logs.log {
    hourly
    missingok
    rotate 24
    compress
    delaycompress
    notifempty
    create 0640 root adm
    postrotate
        invoke-rc.d rsyslog rotate > /dev/null
    endscript
}
```

This configuration will rotate the log file every hour, keep 24 rotated log files, compress the old log files (except for the most recent one), and create a new log file with the specified permissions.

1. In Ubuntu, by default, **`logrotate`** is executed daily by the **`anacron`** service. To make it run hourly, you need to create a new hourly cron job. Create a new script called **`logrotate-hourly`** in **`/etc/cron.hourly/`**:

```bash
sudo nano /etc/cron.hourly/logrotate-hourly
```

Add the following content:

```bash
#!/bin/sh
/usr/sbin/logrotate --state /var/lib/logrotate/logrotate.hourly.status /etc/logrotate.conf
```

Make the script executable:

```bash
sudo chmod +x /etc/cron.hourly/logrotate-hourly
```

Restart the cron service to apply the changes:

```bash
sudo service cron restart
```

Install Python and the necessary libraries:

```bash
sudo apt-get updatesudo apt-get install python3 pip -ypip install Faker
```

## **Step 2: Create the fake log generator script**

Create a file named **`fake_log_generator.py`** and add the following code:

```python
import json
import time
from datetime import datetime
from random import choice, choices
import uuid
from faker import Faker

fake = Faker()

def generate_log_entry():
    service_names = ["Auth", "AppStack", "Database"]
    categories = ["[INFO]", "[WARN]", "[CRITICAL]", "[SECURITY]"]

    with open("clean_words_alpha.txt", "r") as word_file:
        word_list = word_file.read().splitlines()

    log_entry = {
        "id": str(uuid.uuid4()),
        "@timestamp": datetime.now().strftime("%Y-%m-%dT%H:%M:%S.%fZ"),
        "@version": "1.1",
        "message": f"{choice(service_names)} {choice(categories)} {' '.join(choices(word_list, k=5))}",
    }

    return log_entry

def main():
    while True:
        log_entry = generate_log_entry()

        # Load existing log entries
        try:
            with open("fake_logs.log", "r") as log_file:
                log_entries = json.load(log_file)
        except (FileNotFoundError, json.JSONDecodeError):
            log_entries = []

        # Append new log entry and write back to the file
        log_entries.append(log_entry)
        with open("fake_logs.log", "w") as log_file:
            json.dump(log_entries, log_file, indent=2)

        # Sleep for 5 seconds before generating another log entry
        time.sleep(5)

if __name__ == "__main__":
    main()
```

## **Step 3: Download the word list**

Download the word list and save it as **`clean_words_alpha.txt`** in the same directory as the **`fake_log_generator.py`** script:

```bash
wget https://github.com/dwyl/english-words/files/3086945/clean_words_alpha.txt
```

## **Step 4: Create a systemd service**

Create a file named **`fake-log-generator.service`** with the following content:

```makefile
[Unit]
Description=Generate fake logs
After=network.target

[Service]
User=<your-username>
WorkingDirectory=/home/<your-username>/logrotate
ExecStart=/usr/bin/python3 fake_log_generator.py -d logs/
Restart=always

[Install]
WantedBy=multi-user.target
```

Make sure to replace **`/path/to`** with the absolute path to the directory containing the **`fake_log_generator.py`** script.

Reload the systemd daemon, enable, and start the service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable fake_log_generator.service
sudo systemctl start fake_log_generator.service
```

Now the fake log generator script will run reliably as a systemd service, creating log entries in the **`fake_logs.log`** file every 5 seconds.

```bash
export BACALHAU_LOCAL_DIRECTORY_ALLOW_LIST=/home/<your-username>/logrotate/logs
```
