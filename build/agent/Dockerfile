FROM  registry.cn-beijing.aliyuncs.com/aliyun-fc/fc-stable-diffusion:realman-v4
COPY build/agent/agentServer /agent/agent
COPY agent.yaml /agent/agent.yaml
COPY build/agent/entrypoint.sh /docker/entrypoint.sh
EXPOSE 7860