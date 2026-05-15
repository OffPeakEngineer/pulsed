const repositoryUrl = process.env.GITHUB_REPOSITORY
  ? `https://github.com/${process.env.GITHUB_REPOSITORY}.git`
  : 'https://github.com/OffPeakEngineer/psstd.git';

module.exports = {
  branches: ['main'],
  repositoryUrl,
  plugins: [
    '@semantic-release/commit-analyzer',
    '@semantic-release/release-notes-generator',
    [
      '@semantic-release/github',
      {
        assets: [
          { path: 'dist/psstd-linux-amd64', label: 'Linux amd64 binary' },
          { path: 'dist/psstd-linux-arm64', label: 'Linux arm64 binary' },
          { path: 'dist/psstd-darwin-amd64', label: 'macOS Intel binary' },
          { path: 'dist/psstd-darwin-arm64', label: 'macOS Apple Silicon binary' },
          { path: 'dist/psstd-windows-amd64.exe', label: 'Windows amd64 binary' },
          { path: 'dist/psstd-windows-arm64.exe', label: 'Windows arm64 binary' }
        ]
      }
    ]
  ]
};
