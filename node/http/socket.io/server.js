const httpServer = require('http').createServer((req, res) => {
    // serve the index.html file
    res.setHeader('Content-Type', 'text/html');
});

const io = require('socket.io')(httpServer);

io.on('connection', socket => {
    console.log('connect');
});

httpServer.listen(3000, () => {
    console.log('go to http://0.0.0.0:3000');
});