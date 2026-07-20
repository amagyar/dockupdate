#!/usr/bin/env node
// postinstall: downloads the dockupdate release archive matching this
// package version and the current platform from GitHub Releases, verifies
// its SHA256 against the published checksums, and installs the binary next
// to the bin shim.
const crypto = require('crypto');
const fs = require('fs');
const https = require('https');
const http = require('http');
const path = require('path');
const tar = require('tar');

const pkg = require('./package.json');

const REPO = 'adev/dockupdate';
const VERSION = process.env.DOCKUPDATE_VERSION || pkg.version;
const BASE_URL =
  process.env.DOCKUPDATE_BASE_URL ||
  `https://github.com/${REPO}/releases/download/v${VERSION}`;

const PLATFORMS = { darwin: 'darwin', linux: 'linux', win32: 'windows' };
const ARCHES = { x64: 'amd64', arm64: 'arm64' };

function fail(msg) {
  console.error(`dockupdate: ${msg}`);
  process.exit(1);
}

function download(url, dest, redirects = 5) {
  // http is only reachable via the DOCKUPDATE_BASE_URL testing override;
  // the production base URL is always https.
  const transport = url.startsWith('http:') ? http : https;
  return new Promise((resolve, reject) => {
    transport
      .get(url, (res) => {
        if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
          res.resume();
          if (redirects <= 0) return reject(new Error('too many redirects'));
          return resolve(download(res.headers.location, dest, redirects - 1));
        }
        if (res.statusCode !== 200) {
          res.resume();
          return reject(new Error(`GET ${url} -> HTTP ${res.statusCode}`));
        }
        const out = fs.createWriteStream(dest);
        res.pipe(out);
        out.on('finish', () => out.close(resolve));
        out.on('error', reject);
      })
      .on('error', reject);
  });
}

async function main() {
  const goos = PLATFORMS[process.platform];
  const goarch = ARCHES[process.arch];
  if (!goos || !goarch) {
    fail(
      `unsupported platform: ${process.platform}/${process.arch}. ` +
      `supported: darwin/linux/win32 on x64/arm64`
    );
  }

  const archive = `dockupdate_${VERSION}_${goos}_${goarch}.tar.gz`;
  const tmp = fs.mkdtempSync(path.join(require('os').tmpdir(), 'dockupdate-'));

  console.error(`dockupdate: downloading ${archive}`);
  await download(`${BASE_URL}/${archive}`, path.join(tmp, archive));
  await download(`${BASE_URL}/checksums.txt`, path.join(tmp, 'checksums.txt'));

  // Verify checksum.
  const sums = fs.readFileSync(path.join(tmp, 'checksums.txt'), 'utf8');
  const line = sums.split('\n').find((l) => l.trim().endsWith(archive));
  if (!line) fail(`checksums.txt has no entry for ${archive}`);
  const want = line.trim().split(/\s+/)[0];
  const got = crypto
    .createHash('sha256')
    .update(fs.readFileSync(path.join(tmp, archive)))
    .digest('hex');
  if (want !== got) fail(`checksum mismatch for ${archive} (want ${want}, got ${got})`);

  // Extract the binary and place it next to the shim.
  tar.x({ file: path.join(tmp, archive), cwd: tmp, sync: true });
  const extracted = path.join(tmp, goos === 'windows' ? 'dockupdate.exe' : 'dockupdate');
  const dest = path.join(
    __dirname,
    'bin',
    goos === 'windows' ? 'dockupdate.exe' : 'dockupdate-bin'
  );
  fs.copyFileSync(extracted, dest);
  fs.chmodSync(dest, 0o755);
  fs.rmSync(tmp, { recursive: true, force: true });
  console.error(`dockupdate: installed ${VERSION} (${goos}/${goarch})`);
}

main().catch((err) => fail(err.message));
