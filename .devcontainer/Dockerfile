# Changes from upstream Anthropic template (github.com/anthropics/claude-code/tree/main/.devcontainer):
#   - Added: bubblewrap, socat (apt packages — required for Claude Code sandbox mode)
#   - Added: @anthropic-ai/sandbox-runtime to npm global installs (seccomp filters for sandbox)
#   - Added: pre-configured ~/.claude/settings.json with enableWeakerNestedSandbox: true
#     (required for bubblewrap to work inside Docker; volume mount removed in devcontainer.json
#      so this file is not shadowed on first run)
#   - Added: UV installer + PATH for Python toolchain management (interpreter versions,
#     project dependencies, and virtual environments)
#   - Added: chezmoi installer (dotfiles manager, installed to ~/.local/bin)
#   - Removed: go.mod/go.sum pre-cache step — Dev Containers CLI always uses .devcontainer/ as
#     build context regardless of devcontainer.json build.context, so COPY from repo root fails.
#     Go modules are downloaded lazily on first use (proxy.golang.org is in the firewall allowlist).
#   - Note: user plugins (e.g. superpowers) are NOT baked in — they are installed via VS Code
#     dotfiles (chezmoi apply) which runs before the firewall (waitFor: postCreateCommand)

FROM node:20

ARG TZ
ENV TZ="$TZ"

ARG CLAUDE_CODE_VERSION=latest

# Install basic development tools and iptables/ipset
RUN apt-get update && apt-get install -y --no-install-recommends \
  bubblewrap \
  socat \
  less \
  git \
  procps \
  sudo \
  fzf \
  zsh \
  man-db \
  unzip \
  gnupg2 \
  gh \
  iptables \
  ipset \
  iproute2 \
  dnsutils \
  aggregate \
  jq \
  nano \
  vim \
  && apt-get clean && rm -rf /var/lib/apt/lists/*

# Ensure default node user has access to /usr/local/share
RUN mkdir -p /usr/local/share/npm-global && \
  chown -R node:node /usr/local/share

ARG USERNAME=node

# Persist bash history.
RUN SNIPPET="export PROMPT_COMMAND='history -a' && export HISTFILE=/commandhistory/.bash_history" \
  && mkdir /commandhistory \
  && touch /commandhistory/.bash_history \
  && chown -R $USERNAME /commandhistory

# Set `DEVCONTAINER` environment variable to help with orientation
ENV DEVCONTAINER=true

# Create workspace and config directories and set permissions
RUN mkdir -p /workspace /home/node/.claude && \
  chown -R node:node /workspace /home/node/.claude

# Pre-configure Claude Code for sandbox mode inside Docker.
# enableWeakerNestedSandbox is required for bubblewrap to work inside a container.
# The iptables firewall (init-firewall.sh) provides the outer network isolation
# that justifies using the weaker nested sandbox mode.
# Users with a dotfiles manager (chezmoi, etc.) can override this via VS Code's
# dotfiles support, which runs after container creation.
RUN printf '{\n  "sandbox": {\n    "enabled": true,\n    "enableWeakerNestedSandbox": true\n  }\n}\n' \
  > /home/node/.claude/settings.json && \
  chown node:node /home/node/.claude/settings.json

WORKDIR /workspace

ARG GIT_DELTA_VERSION=0.18.2
RUN ARCH=$(dpkg --print-architecture) && \
  wget "https://github.com/dandavison/delta/releases/download/${GIT_DELTA_VERSION}/git-delta_${GIT_DELTA_VERSION}_${ARCH}.deb" && \
  sudo dpkg -i "git-delta_${GIT_DELTA_VERSION}_${ARCH}.deb" && \
  rm "git-delta_${GIT_DELTA_VERSION}_${ARCH}.deb"

# Set up non-root user
USER node

# Install global packages
ENV NPM_CONFIG_PREFIX=/usr/local/share/npm-global
ENV PATH=$PATH:/usr/local/share/npm-global/bin

# Set the default shell to zsh rather than sh
ENV SHELL=/bin/zsh

# Set the default editor and visual
ENV EDITOR=nano
ENV VISUAL=nano

# Default powerline10k theme
ARG ZSH_IN_DOCKER_VERSION=1.2.0
RUN sh -c "$(wget -O- https://github.com/deluan/zsh-in-docker/releases/download/v${ZSH_IN_DOCKER_VERSION}/zsh-in-docker.sh)" -- \
  -p git \
  -p fzf \
  -a "source /usr/share/doc/fzf/examples/key-bindings.zsh" \
  -a "source /usr/share/doc/fzf/examples/completion.zsh" \
  -a "export PROMPT_COMMAND='history -a' && export HISTFILE=/commandhistory/.bash_history" \
  -x

# Install Claude and sandbox-runtime (for seccomp filters)
RUN npm install -g @anthropic-ai/claude-code@${CLAUDE_CODE_VERSION} @anthropic-ai/sandbox-runtime

# Install UV — manages Python toolchain (interpreter versions, project dependencies, virtualenvs)
ARG UV_VERSION=0.10.8
RUN curl -LsSf https://astral.sh/uv/${UV_VERSION}/install.sh | sh
ENV PATH="/home/node/.local/bin:$PATH"

# Install chezmoi — dotfiles manager, available on PATH for VS Code dotfiles hook
RUN sh -c "$(curl -fsLS get.chezmoi.io)" -- -b /home/node/.local/bin

# Install Go (requires root to write to /usr/local)
ARG GO_VERSION=1.23.5
USER root
RUN ARCH=$(dpkg --print-architecture) && \
  curl -fsSL "https://go.dev/dl/go${GO_VERSION}.linux-${ARCH}.tar.gz" \
  | tar -C /usr/local -xzf -
ENV PATH=$PATH:/usr/local/go/bin
ENV GOPATH=/home/node/go
ENV GOBIN=/home/node/go/bin
ENV PATH=$PATH:$GOBIN

# Copy and set up firewall script
COPY init-firewall.sh /usr/local/bin/
USER root
RUN chmod +x /usr/local/bin/init-firewall.sh && \
  echo "node ALL=(root) NOPASSWD: /usr/local/bin/init-firewall.sh" > /etc/sudoers.d/node-firewall && \
  chmod 0440 /etc/sudoers.d/node-firewall
USER node
