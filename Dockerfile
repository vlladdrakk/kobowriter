FROM ubuntu:18.04

RUN apt update && DEBIAN_FRONTEND=noninteractive apt install -y build-essential autoconf automake bison flex gawk libtool libtool-bin libncurses-dev curl file git gperf help2man texinfo unzip wget

RUN useradd -ms /bin/bash ubuntu
USER ubuntu

WORKDIR /home/ubuntu

RUN git clone https://github.com/koreader/koxtoolchain

WORKDIR /home/ubuntu/koxtoolchain

RUN ./gen-tc.sh kobo

WORKDIR /home/ubuntu
RUN wget https://go.dev/dl/go1.18.linux-amd64.tar.gz
RUN tar xzf go1.18.linux-amd64.tar.gz

COPY ./docker-entry.sh /opt/entry.sh

ENTRYPOINT ["/opt/entry.sh"]
