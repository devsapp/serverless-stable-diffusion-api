import random
import cv2
import base64
import requests

class ControlnetRequest:
    def __init__(self, prompt, path):
        self.url = "https://xxxxxxxxxx/txt2img"
        self.prompt = prompt
        self.img_path = path
        self.body = None
        self.L_models = ["xxmix9realistic_v40.safetensors", "chilloutmix_NiPrunedFp16Fix.safetensors",
                         "Realistic_Vision_V2.0.ckpt", "v1-5-pruned-emaonly.ckpt",
                         "墨幽人造人_v1010.safetensors","dreamshaper_7.safetensors", "GuoFeng3_v3.4.safetensors"]
        self.L_lora = ["<lora:blingdbox_v1_mix:1>","<lora:ChinaDollLikeness:1>","<lora:Colorwater_v4:1>","<lora:GachaSpliash4:1>",
                       "<lora:KoreanDollLikeness:1>"]
        self.L_controlnet = ["control_v11p_sd15_canny", "control_v11p_sd15_scribble","control_v11f1p_sd15_depth",
                             "control_v11p_sd15_lineart","control_v11p_sd15_openpose"]

    def select_models(self, L):
        idx = random.randint(0, len(L)-1)
        model = L[idx]
        return model
        # return "xxmix9realistic_v40.safetensors"

    def build_body(self):
        self.body = {
            "stable_diffusion_model": "xxmix9realistic_v40.safetensors",
            "sd_vae": "None",
            "prompt": "'masterpiece, best quality, very detailed, extremely detailed beautiful, super detailed, "
                      "tousled hair, illustration, dynamic angles, girly, fashion clothing, standing, mannequin, "
                      "looking at viewer, interview, beach, beautiful detailed eyes,"
                      " exquisitely beautiful face, floating, high saturation, "
                      "beautiful and detailed light and shadow" + self.select_models(self.L_lora),
            "negative_prompt": "loli,nsfw,logo,text,badhandv4,EasyNegative,ng_deepnegative_v1_75t,"
                               "rev2-badprompt,verybadimagenegative_v1.3,negative_hand-neg,mutated hands and fingers,"
                               "poorly drawn face,extra limb,missing limb,disconnected limbs,malformed hands,ugly",
            "batch_size": 1,
            "steps": 30,
            "cfg_scale": 7,
            "alwayson_scripts": {
                "controlnet": {
                    "args": [
                        {
                            # "image":"images/default/mjDxugVFDr_1.png",
                            "image":self.read_image(),
                            "enabled":True,
                            "module":"canny",
                            "model": "control_v11p_sd15_canny",
                            # "model":self.select_models(self.L_controlnet),
                            "weight":1,
                            "resize_mode":"Crop and Resize",
                            "low_vram":False,
                            "processor_res":512,
                            "threshold_a":100,
                            "threshold_b":200,
                            "guidance_start":0,
                            "guidance_end":1,
                            "pixel_perfect":True,
                            "control_mode":"Balanced",
                            "input_mode":"simple",
                            "batch_images":"",
                            "output_dir":"",
                            "loopback":False
                        }
                    ]
                }
            }
        }

    def send_request(self):
        headers = {"Request-Type":"sync", "content-type": "application/json", "token":"MUEGw51wmaXZhYFpjn5lIUx6nIWnp3enzagiMaWAb1flAZEjbYWa2QFFZ1CxbS6g"}
        response = requests.post(url=self.url, json=self.body, headers=headers)
        return response.json()

    def read_image(self):
        img = cv2.imread(self.img_path)
        retval, bytes = cv2.imencode('.png', img)
        encoded_image = base64.b64encode(bytes).decode('utf-8')
        return encoded_image

    def file_to_base64(self):
        with open(self.img_path, "rb") as file:
            data = file.read()

        base64_str = str(base64.b64encode(data), "utf-8")
        return "data:image/png;base64," + base64_str


if __name__ == '__main__':
    path = '/Users/xxxx/Downloads/mjDxugVFDr_1.png'
    prompt = 'a large avalanche'

    control_net = ControlnetRequest(prompt, path)
    control_net.build_body()
    output = control_net.send_request()
    print(output)