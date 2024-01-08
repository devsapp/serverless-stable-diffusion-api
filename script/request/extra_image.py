import cv2
import base64
import requests

class Request:
    def __init__(self, prompt, path):
        self.url = "https://xxxx/extra_images"
        self.prompt = prompt
        self.img_path = path
        self.body = None

    def build_body(self):
        self.body = {
            # "stable_diffusion_model":"chilloutmix_NiPrunedFp16Fix.safetensors",
            "resize_mode": 0,
            "show_extras_results": True,
            "gfpgan_visibility": 0,
            "codeformer_visibility": 0,
            "codeformer_weight": 0,
            "upscaling_resize": 4,
            # "upscaling_crop": True,
            "upscaler_1": "Lanczos",
            "upscaler_2": "None",
            "extras_upscaler_2_visibility": 0,
            "upscale_first": False,
            # "image":self.file_to_base64(),
            "image" : "images/default/mjDxugVFDr_1.png"
        }


    def send_request(self):
        headers = {"Request-Type":"sync", "content-type": "application/json", "token":"aSV8Ro3qwbMy3qla7fNGORNonmj6KxbvUIVtE5kNl58BwYUUQq0uap6MODxhGqfq"}
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

    control_net = Request(prompt, path)
    control_net.build_body()
    output = control_net.send_request()
    print(output)