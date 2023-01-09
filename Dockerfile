FROM golang:1.19-alpine as build
WORKDIR /
COPY *.go ./
COPY *.mod ./
COPY *.sum ./
RUN go build -o /go-readthenburn-backend

FROM alpine:latest
COPY --from=build /go-readthenburn-backend /
EXPOSE 8080
CMD [ "/go-readthenburn-backend" ]