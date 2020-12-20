FROM paulcager/go-base:latest as build
WORKDIR /go/src/

COPY . /go/src/github.com/paulcager/gb-airspace
RUN cd /go/src/github.com/paulcager/gb-airspace && go test ./... && go install ./...

####################################################################################################


FROM debian:stable-slim
RUN apt-get update && apt-get -y upgrade && apt-get install -y ca-certificates
WORKDIR /app
COPY --from=build /go/bin/* ./
EXPOSE 9092
CMD ["/app/serve-airspace", "--port", ":9092" ]

