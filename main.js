const start = () => {
    const ws = new WebSocket("ws://localhost:8080/ws");
    ws.binaryType = "arraybuffer";

    ws.onmessage = (e) => {
        console.log(e.data);
    };
};

// wss://administered-essay-brochure-controlling.trycloudflare.com/
