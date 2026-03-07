const http = require("http");

const port = process.env.PORT || 8080;
const logLevel = process.env.LOG_LEVEL || "info";

const server = http.createServer((req, res) => {
  if (req.url === "/healthz") {
    res.writeHead(200);
    res.end("ok");
    return;
  }

  res.writeHead(200, { "Content-Type": "application/json" });
  res.end(JSON.stringify({ service: "checkout-api", logLevel }));
});

server.listen(port, () => {
  console.log(`checkout-api listening on ${port} (LOG_LEVEL=${logLevel})`);
});
