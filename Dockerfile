FROM golang:1.25-alpine AS build
WORKDIR /app
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOEXPERIMENT=greenteagc go build -a -installsuffix cgo -o /go-readthenburn-backend ./cmd/server

FROM scratch
COPY --from=build /go-readthenburn-backend /
USER 65534:65534
EXPOSE 8080
CMD ["/go-readthenburn-backend"]