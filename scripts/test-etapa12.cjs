// Teste real do download pelo Chrome. Executar: node scripts/test-etapa12.cjs
const { spawn, execFileSync } = require("node:child_process");
const fs = require("node:fs");
const path = require("node:path");
const net = require("node:net");
const assert = require("node:assert/strict");
const root = path.resolve(__dirname, "..");
const dir = fs.mkdtempSync("/tmp/aula-sync-etapa12-");
const children = [];
const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
let ws;
async function port() {
  const s = net.createServer();
  await new Promise((resolve) => s.listen(0, "127.0.0.1", resolve));
  const p = s.address().port;
  await new Promise((resolve) => s.close(resolve));
  return p;
}
function start(cmd, args, options = {}) {
  const child = spawn(cmd, args, {
    detached: true,
    stdio: "ignore",
    ...options,
  });
  child.on("error", (e) => {
    console.error(e);
    process.exitCode = 1;
  });
  children.push(child);
  return child;
}
async function ready(url) {
  for (let i = 0; i < 200; i++) {
    try {
      const r = await fetch(url);
      if (r.ok) return await r.json().catch(() => null);
    } catch {}
    await sleep(100);
  }
  throw Error(`Servidor indisponível: ${url}`);
}
async function run() {
  const apiPort = await port(),
    vitePort = await port(),
    chromePort = await port();
  const base = `http://127.0.0.1:${apiPort}`;
  const bin = path.join(dir, "api");
  execFileSync("go", ["build", "-o", bin, "./cmd/api"], {
    cwd: path.join(root, "backend"),
  });
  start(bin, [], {
    env: {
      ...process.env,
      PORT: String(apiPort),
      DATABASE_PATH: path.join(dir, "teste.db"),
      GOOGLE_CLIENT_ID: "",
      GOOGLE_CLIENT_SECRET: "",
      GOOGLE_REDIRECT_URI: "",
      APP_URL: "",
    },
  });
  start(
    "node",
    [
      "node_modules/vite/bin/vite.js",
      "--host",
      "127.0.0.1",
      "--port",
      String(vitePort),
      "--strictPort",
    ],
    {
      cwd: path.join(root, "frontend"),
      env: { ...process.env, API_PROXY_TARGET: base },
    },
  );
  start(process.env.CHROME_BIN || "google-chrome", [
    "--headless=new",
    "--no-sandbox",
    "--disable-gpu",
    `--remote-debugging-port=${chromePort}`,
    `--user-data-dir=${path.join(dir, "chrome")}`,
    "about:blank",
  ]);
  await ready(`${base}/health`);
  await ready(`http://127.0.0.1:${vitePort}`);
  async function request(route, data) {
    const r = await fetch(base + route, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(data),
    });
    assert(r.ok, await r.text());
  }
  await request("/turmas", { nome: "Turma PDF" });
  for (let i = 0; i < 21; i++)
    await request("/aulas", {
      data: "2026-10-01",
      inicio: "08:00",
      fim: "09:00",
      turma_id: 1,
      conteudo: `Leitura e revisão ${i}`,
    });
  const targets = await ready(`http://127.0.0.1:${chromePort}/json`);
  ws = new WebSocket(
    targets.find((t) => t.type === "page").webSocketDebuggerUrl,
  );
  await new Promise((resolve, reject) => {
    ws.onopen = resolve;
    ws.onerror = reject;
  });
  let id = 0;
  const pending = new Map();
  const requests = [];
  function cdp(method, params = {}) {
    return new Promise((resolve, reject) => {
      const n = ++id;
      pending.set(n, { resolve, reject });
      ws.send(JSON.stringify({ id: n, method, params }));
    });
  }
  ws.onmessage = (e) => {
    const r = JSON.parse(e.data);
    if (
      r.method === "Network.requestWillBeSent" &&
      r.params.request.url.includes("/aulas/relatorio.pdf")
    )
      requests.push(r.params.request.url);
    if (r.id) {
      const p = pending.get(r.id);
      pending.delete(r.id);
      r.error ? p.reject(r.error) : p.resolve(r.result);
    }
  };
  async function evaluate(expression) {
    const r = await cdp("Runtime.evaluate", {
      expression,
      awaitPromise: true,
      returnByValue: true,
    });
    if (r.exceptionDetails) throw Error(JSON.stringify(r.exceptionDetails));
    return r.result.value;
  }
  async function waitFor(expression) {
    for (let i = 0; i < 150; i++) {
      if (await evaluate(expression)) return;
      await sleep(100);
    }
    throw Error(`Timeout: ${expression}`);
  }
  async function click(text) {
    await waitFor(
      `Array.from(document.querySelectorAll('button')).some(b => b.textContent.trim() === ${JSON.stringify(text)} && !b.disabled)`,
    );
    await evaluate(
      `Array.from(document.querySelectorAll('button')).find(b => b.textContent.trim() === ${JSON.stringify(text)}).click()`,
    );
  }
  await cdp("Network.enable");
  // Google indisponível: o relatório deve continuar funcionando.
  await cdp("Network.setBlockedURLs", {
    urls: ["*googleapis.com*", "*accounts.google.com*"],
  });
  await cdp("Browser.setDownloadBehavior", {
    behavior: "allow",
    downloadPath: dir,
  });
  await cdp("Page.navigate", { url: `http://127.0.0.1:${vitePort}` });
  await waitFor("!!document.querySelector('#aba-historico')");
  await click("Histórico");
  await waitFor("document.body.innerText.includes('21 aula(s) no filtro')");
  await click("Próxima");
  await waitFor("document.body.innerText.includes('Página 2 de 2')");
  await click("Exportar PDF");
  await waitFor("document.body.innerText.includes('PDF gerado.')");
  for (
    let i = 0;
    i < 100 && !fs.existsSync(path.join(dir, "relatorio-aulas.pdf"));
    i++
  )
    await sleep(100);
  const pdf = path.join(dir, "relatorio-aulas.pdf");
  assert(fs.readFileSync(pdf).subarray(0, 5).equals(Buffer.from("%PDF-")));
  const text = execFileSync("pdftotext", [pdf, "-"], { encoding: "utf8" });
  assert(text.includes("21 aula(s)") && text.includes("21h00"));
  assert(!new URL(requests[0]).searchParams.has("pagina"));
  // Um rascunho no formulário de filtros não muda os filtros já aplicados.
  await evaluate(
    "const e=document.querySelector('#data-inicial');e.value='2027-01-01';e.dispatchEvent(new Event('input',{bubbles:true}));",
  );
  await click("Exportar PDF");
  await waitFor("document.body.innerText.includes('PDF gerado.')");
  assert(!new URL(requests.at(-1)).searchParams.has("data_inicial"));
  await click("Filtrar");
  await waitFor("document.body.innerText.includes('0 aula(s) no filtro')");
  await click("Exportar PDF");
  await waitFor("document.body.innerText.includes('PDF gerado.')");
  assert.equal(
    new URL(requests.at(-1)).searchParams.get("data_inicial"),
    "2027-01-01",
  );
  await cdp("Network.emulateNetworkConditions", {
    offline: true,
    latency: 0,
    downloadThroughput: 0,
    uploadThroughput: 0,
  });
  await click("Exportar PDF");
  await waitFor(
    "document.body.innerText.includes('Não foi possível exportar o PDF.')",
  );
  assert(
    await evaluate(
      "!Array.from(document.querySelectorAll('button')).find(b=>b.textContent.trim()==='Exportar PDF').disabled",
    ),
  );
  await cdp("Network.emulateNetworkConditions", {
    offline: false,
    latency: 0,
    downloadThroughput: -1,
    uploadThroughput: -1,
  });
  await click("Exportar PDF");
  await waitFor("document.body.innerText.includes('PDF gerado.')");
  await cdp("Emulation.setDeviceMetricsOverride", {
    width: 375,
    height: 812,
    deviceScaleFactor: 1,
    mobile: true,
  });
  assert(
    await evaluate("document.documentElement.scrollWidth <= window.innerWidth"),
  );
  const shot = await cdp("Page.captureScreenshot", {
    format: "png",
    captureBeyondViewport: true,
  });
  fs.writeFileSync(
    path.join(dir, "historico.png"),
    Buffer.from(shot.data, "base64"),
  );
  console.log(
    "PASS: download real, todas as páginas, filtros aplicados, vazio, Google bloqueado, falha de rede/recuperação e largura de 375px.",
  );
  console.log(`Artefatos: ${dir}`);
}
run()
  .catch((e) => {
    console.error(e);
    process.exitCode = 1;
  })
  .finally(async () => {
    ws?.close();
    for (const child of children.reverse()) {
      try {
        process.kill(-child.pid, "SIGTERM");
      } catch {}
    }
    await sleep(500);
  });
