FROM golang:1.16.2-buster as builder

WORKDIR /build
COPY go.mod go.sum main.go ./
RUN go build .

FROM gcr.io/cloud-spanner-emulator/emulator:1.2.0 as runtime
COPY --from=builder /build/spanner-emulator ./

EXPOSE 9010 9020

ENTRYPOINT ["./spanner-emulator"]
