FROM paulcager/go-base:latest AS build
WORKDIR /go/src/

COPY . /go/src/github.com/paulcager/gb-airspace
RUN cd /go/src/github.com/paulcager/gb-airspace && go test ./... && CGO_ENABLED=0 go install ./...

####################################################################################################


FROM scratch
WORKDIR /app
COPY --from=build /go/bin/* ./
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
EXPOSE 9092
CMD ["/app/serve-airspace", "--port", ":9092" ]

