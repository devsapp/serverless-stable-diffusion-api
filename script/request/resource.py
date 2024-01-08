import requests
import json
url = "http://localhost:7860/batch_update_sd_resource"
headers = {"Request-Type":"sync", "content-type": "application/json", "token":"MUEGw51wmaXZhYFpjn5lIUx6nIWnp3enzagiMaWAb1flAZEjbYWa2QFFZ1CxbS6g"}
s = json.dumps({
    "models":[
        "xxmix9realistic_v40.safetensors"
    ],
    "extraArgs":"--api --nowebui",
    "vpcConfig":{
        "securityGroupId":"xxx",
        "vSwitchIds":[
            "xxx"
        ],
        "vpcId":"xxx"
    }
})
r = requests.post(url, data=s, headers=headers)
print(r.json())