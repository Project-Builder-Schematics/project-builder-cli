'use strict';

/**
 * Hello World schematic — creates hello.txt in the workspace root.
 * Uses CommonJS to avoid needing a transpilation step.
 *
 * This is a minimal schematic for integration testing (REQ-21.1).
 */
const { Rule } = require('@angular-devkit/schematics');

function hello(_options) {
  return (tree, _context) => {
    tree.create('hello.txt', 'Hello, world!\n');
    return tree;
  };
}

module.exports = { hello };
