import cv2
import base64
import requests


class ControlnetRequest:
    def __init__(self, prompt, path):
        self.url = "https://xxxxxxxxxxx/img2img"
        self.prompt = prompt
        self.img_path = path
        self.body = None

    def build_body(self):
        self.body = {
            "stable_diffusion_model": "xxmix9realistic_v40.safetensors",
            "sd_vae": "None",
            "prompt": "product place on table,((kitchen)), food, snacks,photorealistic, realistic, photography, masterpiece, best quality, no human",
            "negative_prompt": "unrealistic, poor texture, poor quality,clear edges, bad material, soft detail, bad pictures, (bad hand), ((low quality)), ((worst quality)), nsfw",
            "width": 512,
            "height": 512,
            "seed": 112233,
            "steps": 30,
            "cfg_scale": 7,
            "sampler_name": "Euler a",
            "batch_size": 1,
            "init_images": [
                # "images/default/8A8088848B128072018B1280744F0001.png"
                self.file_to_base64()
            ],
            # "mask": "images/default/8A8088848B128072018B1280744F0001.png",
            "mask": self.file_to_base64(),
            "mask_blur": 4,
            "inpainting_mask_invert": 1,
            "denoising_strength": 1,
            "inpainting_fill": 2,
            "inpaint_full_res": False,
            "inpaint_full_res_padding": 32,
            "resize_mode": 0,
            "override_settings": {},
            "alwayson_scripts": {
                "controlnet": {
                    "args": [
                        {
                            "enabled":True,
                            "module": "canny",
                            "model": "control_v11p_sd15_canny",
                            "weight": 1,
                            # "image": "images/default/8A8088848B128072018B1280744F0001.png",
                            "image": self.file_to_base64(),
                            "resize_mode": 0,
                            "rgbbgr_mode": False,
                            "lowvram": False,
                            "processor_res": 512,
                            "threshold_a": 100,
                            "threshold_b": 200,
                            "guidance_start": 0,
                            "guidance_end": 1,
                            "control_mode": 0,
                            "pixel_perfect": True
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
        # return base64_str


if __name__ == '__main__':
    path = '/Users/xxxx/Desktop/8A8088848B128072018B1280744F0001.png'
    prompt = 'a large avalanche'

    control_net = ControlnetRequest(prompt, path)
    control_net.build_body()
    output = control_net.send_request()
    print(output)