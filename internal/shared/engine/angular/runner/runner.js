// runner.js — embedded Node.js runner for AngularSubprocessAdapter.
//
// # Protocol
//
// This script bridges @angular-devkit/schematics-cli to the Go adapter via NDJSON.
// Each event is one JSON line on stdout. Stdin carries:
//   Line 1: JSON object containing schematic inputs (map[string]any from Go).
//   Subsequent lines: input_reply messages from the Go adapter.
//
// # Invocation
//
//   node runner.js --collection <collection> --schematic <name>
//
// # Event types (all 12 in the sealed catalogue)
//
//   file_created    — tree.create() or tree.mkdir()
//   file_modified   — tree.overwrite()
//   file_deleted    — tree.delete()
//   script_started  — lifecycle script begins
//   script_stopped  — lifecycle script ends
//   log_line        — logger output (info/warn/error/debug)
//   input_requested — schematic prompt to the user
//   input_provided  — echo of the input reply (emitted by Go adapter, not runner)
//   progress        — incremental progress update
//   done            — terminal success event
//   failed          — terminal error event
//   cancelled       — terminal cancellation (emitted by Go adapter, not runner)
//
// # Security
//
//   NO shell invocations. The runner is exec'd directly by Node.
//   Inputs arrive via stdin JSON, not argv.
//   Sensitive values are never printed to stderr.
//
'use strict';

const { promisify } = require('util');
const readline = require('readline');
const path = require('path');

// --- argument parsing ---

const args = process.argv.slice(2);
let collection = '';
let schematicName = '';

for (let i = 0; i < args.length; i++) {
  if (args[i] === '--collection' && i + 1 < args.length) {
    collection = args[++i];
  } else if (args[i] === '--schematic' && i + 1 < args.length) {
    schematicName = args[++i];
  }
}

if (!collection || !schematicName) {
  emit({ type: 'failed', seq: 1, message: 'runner: missing --collection or --schematic argument' });
  process.exit(1);
}

// --- NDJSON emit ---

let seq = 0;
function emit(obj) {
  seq++;
  obj.seq = seq;
  obj.at = new Date().toISOString();
  process.stdout.write(JSON.stringify(obj) + '\n');
}

// --- stdin reading ---

// Read all stdin lines. Line 1 is the inputs JSON; subsequent lines are input_reply messages.
const stdinLines = [];
const rl = readline.createInterface({ input: process.stdin, crlfDelay: Infinity });

rl.on('line', line => stdinLines.push(line));

rl.once('close', () => {
  // Parse the first line as inputs JSON.
  let inputs = {};
  if (stdinLines.length > 0) {
    try {
      inputs = JSON.parse(stdinLines[0]) || {};
    } catch (e) {
      // Empty or malformed inputs — use empty object.
      inputs = {};
    }
  }

  runSchematic(collection, schematicName, inputs).catch(err => {
    emit({ type: 'failed', message: err && err.message ? err.message : String(err) });
    process.exit(1);
  });
});

// --- schematic execution ---

async function runSchematic(collectionPath, schName, inputs) {
  let schematics;
  try {
    schematics = require('@angular-devkit/schematics');
  } catch (e) {
    emit({ type: 'failed', message: 'Cannot find @angular-devkit/schematics — is it installed? ' + e.message });
    process.exit(1);
  }

  let schematicsTools;
  try {
    schematicsTools = require('@angular-devkit/schematics/tools');
  } catch (e) {
    emit({ type: 'failed', message: 'Cannot find @angular-devkit/schematics/tools — is it installed? ' + e.message });
    process.exit(1);
  }

  const { SchematicEngine } = schematics;
  const { FileSystemEngineHost, NodeWorkflow } = schematicsTools;

  // Resolve collection path (relative paths resolved from cwd).
  const resolvedCollection = path.isAbsolute(collectionPath)
    ? collectionPath
    : path.resolve(process.cwd(), collectionPath);

  const workflow = new NodeWorkflow(process.cwd(), {
    force: false,
    dryRun: false,
    packageManager: 'npm',
    resolvePaths: [process.cwd()],
    schemaValidation: true,
  });

  // Track progress.
  let stepCount = 0;
  let fileCount = 0;

  // Subscribe to the lifecycle reporter.
  workflow.reporter.subscribe(report => {
    const { kind, path: filePath, content } = report;
    switch (kind) {
      case 'create':
        fileCount++;
        emit({ type: 'file_created', path: filePath, is_dir: false });
        break;
      case 'update':
      case 'overwrite':
        emit({ type: 'file_modified', path: filePath });
        break;
      case 'delete':
        emit({ type: 'file_deleted', path: filePath });
        break;
      case 'rename':
        // Treat rename as delete old + create new.
        emit({ type: 'file_deleted', path: report.to || filePath });
        emit({ type: 'file_created', path: filePath, is_dir: false });
        break;
      default:
        break;
    }
    stepCount++;
    emit({ type: 'progress', step: stepCount, total: 0, label: kind + ' ' + filePath });
  });

  // Subscribe to the lifecycle logger.
  workflow.logger.subscribe(entry => {
    const level = entry.level || 'info';
    emit({
      type: 'log_line',
      level: String(level),
      source: 'stdout',
      text: entry.message,
      sensitive: false,
    });
  });

  try {
    await workflow.execute({
      collection: resolvedCollection,
      schematic: schName,
      options: inputs,
      allowPrivate: false,
      debug: false,
      logger: undefined, // use workflow.logger
    }).toPromise();

    emit({ type: 'done' });
  } catch (err) {
    const msg = err && err.message ? err.message : String(err);
    emit({ type: 'failed', message: msg });
    process.exit(1);
  }
}
