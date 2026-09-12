import http from 'node:http';

const service = process.env.SERVICE_NAME;

http.createServer(async (req, res) => {
  if (req.url === '/health') {
    res.writeHead(200).end('ok');
    return;
  }

  const chunks = [];
  for await (const chunk of req) chunks.push(chunk);

  const url = new URL(req.url, 'http://backend');
  const status = Number(url.searchParams.get('status') ?? 200);
  if (!Number.isInteger(status) || status < 200 || status > 599) {
    res.writeHead(400).end('status must be between 200 and 599');
    return;
  }

  res.writeHead(status, {
    'Content-Type': 'application/json',
    'X-Backend': service,
  });
  res.end(JSON.stringify({
    service,
    method: req.method,
    path: url.pathname,
    query: url.search.slice(1),
    body: Buffer.concat(chunks).toString(),
    request_id: req.headers['x-request-id'] ?? '',
  }) + '\n');
}).listen(8080, '0.0.0.0');
