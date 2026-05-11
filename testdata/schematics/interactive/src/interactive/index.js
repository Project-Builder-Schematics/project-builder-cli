'use strict';

/**
 * Interactive schematic — prompts for a name, creates greeting.txt.
 * For integration testing (REQ-21, REQ-09).
 */

function interactive(options) {
  return (tree, _context) => {
    const name = options.name || 'world';
    tree.create('greeting.txt', `Hello, ${name}!\n`);
    return tree;
  };
}

module.exports = { interactive };
