FROM ubuntu:22.04

ADD huawei-sync /usr/bin

ADD env.sh /usr/bin

RUN apt update && apt install -y skopeo && chmod +x /usr/bin/env.sh && chmod +x /usr/bin/huawei-sync

ENTRYPOINT ["env.sh"]