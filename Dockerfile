FROM golang:1.25-alpine AS build

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN GOOS=js GOARCH=wasm go build -o /out/web/game.wasm ./cmd/client
RUN cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" /out/web/wasm_exec.js
RUN CGO_ENABLED=0 go build -o /out/packov-server ./cmd/server
RUN cp -R web/* /out/web/ && cp -R content /out/content

FROM alpine:3.22

WORKDIR /app
COPY --from=build /out/packov-server /app/packov-server
COPY --from=build /out/web /app/web
COPY --from=build /out/content /app/content
ENV ADDR=:8080
EXPOSE 8080
CMD ["/app/packov-server"]
