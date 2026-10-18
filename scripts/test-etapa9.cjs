// Teste de integração no Chrome real; usa apenas módulos nativos do Node 22.
// Executar da raiz: node scripts/test-etapa9.cjs
const { spawn, execFileSync } = require("node:child_process");
const fs = require("node:fs");
const path = require("node:path");
const net = require("node:net");
const assert = require("node:assert/strict");
const root = path.resolve(__dirname, "..");
const dir = fs.mkdtempSync("/tmp/aula-sync-etapa9-");
const children = [];
const sleep = (ms) => new Promise((resolve) => setTimeout(resolve, ms));
let ws;
async function port() {
  const server = net.createServer();
  await new Promise((resolve, reject) => {
    server.once("error", reject);
    server.listen(0, "127.0.0.1", resolve);
  });
  const value = server.address().port;
  await new Promise((resolve) => server.close(resolve));
  return value;
}
function start(command, args, options = {}) {
  const child = spawn(command, args, {
    detached: true,
    stdio: ["ignore", "ignore", "pipe"],
    ...options,
  });
  child.on("error", (error) => {
    console.error(error);
    process.exitCode = 1;
  });
  child.stderr.on("data", () => {});
  children.push(child);
  return child;
}
async function ready(url) {
  for (let i = 0; i < 100; i++) {
    try {
      const r = await fetch(url);
      if (r.ok) return await r.json().catch(() => null);
    } catch {}
    await sleep(100);
  }
  throw Error(`Servidor não iniciou: ${url}`);
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
  const api = start(bin, [], {
    env: {
      ...process.env,
      PORT: String(apiPort),
      DATABASE_PATH: path.join(dir, "teste.db"),
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
  function cdp(method, params = {}) {
    return new Promise((resolve, reject) => {
      const n = ++id;
      pending.set(n, { resolve, reject });
      ws.send(JSON.stringify({ id: n, method, params }));
    });
  }
  ws.onmessage = (event) => {
    const value = JSON.parse(event.data);
    if (value.method === "Page.javascriptDialogOpening")
      void cdp("Page.handleJavaScriptDialog", { accept: true });
    if (value.id) {
      const p = pending.get(value.id);
      pending.delete(value.id);
      value.error ? p.reject(value.error) : p.resolve(value.result);
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
    for (let i = 0; i < 100; i++) {
      if (await evaluate(expression)) return;
      await sleep(100);
    }
    throw Error(
      `Timeout: ${expression}\n${await evaluate("document.body.innerText")}`,
    );
  }
  async function set(selector, value) {
    await evaluate(
      `(()=>{const e=document.querySelector(${JSON.stringify(selector)});e.value=${JSON.stringify(value)};e.dispatchEvent(new Event('input',{bubbles:true}));e.dispatchEvent(new Event('change',{bubbles:true}));})()`,
    );
  }
  async function click(label) {
    await evaluate(
      `(()=>{const e=[...document.querySelectorAll('button')].find(e=>e.textContent.trim()===${JSON.stringify(label)});if(!e||e.disabled)throw Error('Botão indisponível: '+${JSON.stringify(label)});e.click()})()`,
    );
  }
  async function total(value) {
    await waitFor(
      `document.body.innerText.includes(${JSON.stringify("Carga horária: " + value)}) && !document.body.innerText.includes('Carregando histórico…')`,
    );
  }
  async function request(method, route, body, status = 200) {
    const response = await fetch(base + route, {
      method,
      headers: { "Content-Type": "application/json" },
      ...(body ? { body: JSON.stringify(body) } : {}),
    });
    assert.equal(response.status, status);
    return response.json();
  }
  await cdp("Page.enable");
  await cdp("Page.navigate", { url: `http://127.0.0.1:${vitePort}` });
  await waitFor(`document.body.innerText.includes('Nenhuma aula encontrada')`);
  await evaluate(`document.querySelector('details').open=true`);
  await set("#novo-nome", "Augusto");
  await click("Criar e selecionar");
  await waitFor(`document.querySelector('#turma').value==='1'`);
  await set("#data", "2026-10-18");
  await set("#inicio", "08:00");
  await set("#fim", "09:00");
  await set("#conteudo", "Conteúdo inicial");
  await click("Salvar aula");
  await total("1h00");
  await click("Editar");
  await waitFor(
    `document.querySelector('#form-title').textContent==='Editar aula'`,
  );
  await set("#fim", "10:00");
  await click("Salvar alterações");
  await total("2h00");
  let aula = await request("GET", "/aulas/1");
  assert.equal(aula.versao, 2);
  assert.equal(aula.duracao_minutos, 120);
  await click("Editar");
  await set("#fim", "07:00");
  await click("Salvar alterações");
  await waitFor(
    `document.body.innerText.includes('O término deve ser posterior')`,
  );
  assert.equal((await request("GET", "/aulas/1")).versao, 2);
  // Simular outra aba que salvou a versão aberta no formulário.
  const corpo = {
    data: aula.data,
    inicio: aula.inicio,
    fim: aula.fim,
    turma_id: aula.turma_id,
    conteudo: "Alterado em outra aba",
    observacoes: "",
    intervalo_minutos: 0,
    versao: aula.versao,
  };
  await request("PUT", "/aulas/1", corpo);
  await set("#fim", "11:00");
  await set("#conteudo", "Minha edição preservada");
  await click("Salvar alterações");
  await waitFor(
    `document.body.innerText.includes('Esta aula mudou desde que foi aberta')`,
  );
  assert.equal(
    await evaluate(`document.querySelector('#conteudo').value`),
    "Minha edição preservada",
  );
  assert.equal(
    (await request("GET", "/aulas/1")).conteudo,
    "Alterado em outra aba",
  );
  await click("Fechar edição");
  await waitFor(
    `document.querySelector('.aulas')?.textContent.includes('Alterado em outra aba')`,
  );
  await click("Cancelar aula");
  await total("0h00");
  await set("#situacao", "canceladas");
  await click("Filtrar");
  await waitFor(
    `document.querySelector('.aulas')?.textContent.includes('Cancelada')`,
  );
  assert.equal(
    await evaluate(
      `[...document.querySelectorAll('.aulas button')].some(e=>e.textContent==='Editar')`,
    ),
    false,
  );
  await click("Reativar aula");
  await waitFor(`document.body.innerText.includes('Nenhuma aula encontrada')`);
  await click("Limpar");
  await total("2h00");
  // Cancelar o único registro da última página deve voltar para a página anterior.
  for (let i = 0; i < 20; i++) {
    const { versao, ...novo } = corpo;
    await request("POST", "/aulas", novo, 201);
  }
  await click("Filtrar");
  await waitFor(`document.body.innerText.includes('21 aula(s)')`);
  await click("Próxima");
  await waitFor(`document.body.innerText.includes('Página 2 de 2')`);
  await click("Cancelar aula");
  await waitFor(
    `document.body.innerText.includes('Página 1 de 1') && document.querySelectorAll('.aulas li').length===20`,
  );
  await cdp("Emulation.setDeviceMetricsOverride", {
    width: 375,
    height: 900,
    deviceScaleFactor: 1,
    mobile: true,
  });
  assert(
    await evaluate("document.documentElement.scrollWidth <= window.innerWidth"),
  );
  await evaluate(`document.querySelector('#historico-title').scrollIntoView()`);
  const shot = await cdp("Page.captureScreenshot", { format: "png" });
  fs.writeFileSync(
    path.join(dir, "historico.png"),
    Buffer.from(shot.data, "base64"),
  );
  await cdp("Input.dispatchKeyEvent", {
    type: "keyDown",
    key: "Tab",
    code: "Tab",
    windowsVirtualKeyCode: 9,
  });
  await cdp("Input.dispatchKeyEvent", {
    type: "keyUp",
    key: "Tab",
    code: "Tab",
    windowsVirtualKeyCode: 9,
  });
  assert(await evaluate("document.activeElement!==document.body"));
  await click("Editar");
  await set("#conteudo", "Preservar durante falha");
  api.kill("SIGTERM");
  await sleep(500);
  await click("Salvar alterações");
  await waitFor(
    `document.body.innerText.includes('Os campos foram preservados')`,
  );
  assert.equal(
    await evaluate(`document.querySelector('#conteudo').value`),
    "Preservar durante falha",
  );
  start(bin, [], {
    env: {
      ...process.env,
      PORT: String(apiPort),
      DATABASE_PATH: path.join(dir, "teste.db"),
    },
  });
  await ready(`${base}/health`);
  await click("Salvar alterações");
  await waitFor(
    `document.querySelector('.aulas')?.textContent.includes('Preservar durante falha')`,
  );
  if (process.argv.includes("--google")) {
    const link =
      process.env.GOOGLE_TEST_SHEET_URL ||
      "https://docs.google.com/spreadsheets/d/teste-planilha-A/edit?usp=sharing";
    const canonica = new URL(link);
    canonica.search = "";
    canonica.hash = "";
    await waitFor(
      `document.body.innerText.includes('Nenhuma planilha configurada.')`,
    );
    await set("#google-url", link);
    await click("Salvar destino");
    await waitFor(
      `document.body.innerText.includes('Destino salvo localmente.')`,
    );
    const config = await request("GET", "/integracoes/google/configuracao");
    assert.equal(config.destino.url, canonica.toString());
    assert.equal(config.versao, 1);
    // Aguarda o novo documento para não confundir o formulário antigo com
    // a configuração carregada após o reload.
    await evaluate("window.__googleAntesReload = true");
    await cdp("Page.reload");
    await waitFor(
      `!window.__googleAntesReload && document.querySelector('#google-url')?.value===${JSON.stringify(canonica.toString())} && !document.querySelector('#google-url').matches(':disabled')`,
    );
    await request("PUT", "/integracoes/google/configuracao", {
      url: "https://docs.google.com/spreadsheets/d/teste-planilha-B/edit",
      versao: 1,
    });
    await set("#google-url", link);
    await click("Salvar destino");
    await waitFor(
      `document.body.innerText.includes('A configuração mudou em outra aba')`,
    );
    assert.equal(
      await evaluate(`document.querySelector('#google-url').value`),
      link,
    );
    await click("Recarregar configuração e prévia");
    await waitFor(
      `document.querySelector('#google-url').value.includes('teste-planilha-B')`,
    );
    await set("#google-url", "https://example.com/planilha");
    await click("Salvar destino");
    await waitFor(`document.body.innerText.includes('Informe um link HTTPS')`);
    await set("#google-url", link);
    await click("Salvar destino");
    await waitFor(
      `document.body.innerText.includes('Destino salvo localmente.')`,
    );
    for (const data of ["2026-11-01", "2027-10-01"]) {
      const { versao, ...novo } = corpo;
      await request("POST", "/aulas", { ...novo, data }, 201);
    }
    await click("Recarregar configuração e prévia");
    await waitFor(
      `document.body.innerText.includes('Registros aulas - 10/2027') && document.body.innerText.includes('Registros aulas - 11/2026') && document.body.innerText.includes('Registros aulas - 10/2026')`,
    );
    assert(
      await evaluate(
        "document.documentElement.scrollWidth <= window.innerWidth",
      ),
    );
    await evaluate(`document.querySelector('#google-title').scrollIntoView()`);
    const googleShot = await cdp("Page.captureScreenshot", { format: "png" });
    fs.writeFileSync(
      path.join(dir, "google-config.png"),
      Buffer.from(googleShot.data, "base64"),
    );
    const plano = await request("GET", "/integracoes/google/planejamento");
    assert.equal(plano.verificacao_remota, "nao_realizada");
    assert.equal(plano.configuracao.destino.id, config.destino.id);
    console.log(
      "PASS: configuração Google local, link normalizado, persistência, conflito, troca de destino, validação e prévia mensal. Nenhuma chamada ao Google.",
    );
  }
  console.log(
    "PASS: cadastro, edição, recálculo, validação, conflito, cancelamento, reativação, paginação, celular, foco e recuperação.",
  );
  console.log(`Artefatos temporários: ${dir}`);
}
run()
  .catch((error) => {
    console.error(error);
    process.exitCode = 1;
  })
  .finally(async () => {
    ws?.close();
    for (const child of children.reverse()) {
      if (!child.pid) continue;
      try {
        process.kill(-child.pid, "SIGTERM");
      } catch {}
    }
    await sleep(500);
  });
