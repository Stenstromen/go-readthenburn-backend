FROM golang:1.23-alpine as build
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o /go-readthenburn-backend ./cmd/server

FROM scratch
COPY --from=build /go-readthenburn-backend /
USER 65534:65534
EXPOSE 8080
CMD ["/go-readthenburn-backend"]