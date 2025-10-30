# Changesets

This directory contains changeset files for managing versions and changelogs.

## How to use

When you make changes to the Node.js package that should be included in a release, run:

```bash
npm run changeset
```

Follow the prompts to:
1. Select the type of change (major, minor, patch)
2. Provide a summary of the changes

The changeset will be committed with your PR. When ready to release:

```bash
npm run version  # Updates version and CHANGELOG
npm run release  # Publishes to npm
```

Learn more at https://github.com/changesets/changesets
