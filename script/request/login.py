import json
import requests

url = "https://localhost:8000/login"
header = {"Request-Type": "sync", "Task-Flag": "true"}
data={
    "UserName":"xxxx",
    "Password":"xxxx"
}
r = requests.post(url, data=json.dumps(data))
print(r.content)