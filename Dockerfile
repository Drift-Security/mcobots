FROM golang
ADD . /go/src/github.com/stroncium/mcobots
RUN go get github.com/stroncium/discordgo
RUN go get github.com/stroncium/gg
RUN go get github.com/pdepip/go-binance/binance
RUN go get github.com/fogleman/gg
RUN go get github.com/jteeuwen/go-bindata/...

WORKDIR /go/src/github.com/stroncium/mcobots
RUN go generate
RUN go install ./...
