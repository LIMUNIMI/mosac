# BASE IMAGE FOR JUCE PLUGIN COMPILING
FROM --platform=linux/amd64 ubuntu:22.04 as go_juce_builder

ENV DEBIAN_FRONTEND=noninteractive

RUN apt-get update && apt-get install -y \
  git \
  apt-utils \
  build-essential \
  cmake \
  ninja-build \
  pkg-config \
  libwebkit2gtk-4.0-dev \
  libgtk-3-dev \
  libfontconfig1-dev \
  libfreetype6-dev \
  libx11-dev \
  libxinerama-dev \
  libxrandr-dev \
  libxcursor-dev \
  mesa-common-dev \
  libgl1-mesa-dev \
  libasound2-dev \
  libjack-jackd2-dev \
  libcurl4-openssl-dev \
  && rm -rf /var/lib/apt/lists/*

WORKDIR /workspace



# LINUX STAGE
FROM go_juce_builder AS go_juce_builder_linux

RUN apt-get update && apt-get install -y \
  binutils-gold \
  lv2-dev \
  && rm -rf /var/lib/apt/lists/*

CMD ["bash", "-c", "echo 'Linux container started.'"]



# WINDOWS STAGE
FROM go_juce_builder as go_juce_builder_windows

RUN apt-get update && apt-get install -y \
  curl \
  ca-certificates \
  xz-utils \
  && rm -rf /var/lib/apt/lists/*

RUN curl --fail -L https://github.com/llvm/llvm-project/releases/download/llvmorg-19.1.0/LLVM-19.1.0-Linux-X64.tar.xz | \
  tar -xJ -C /usr/local --strip-components=1 

RUN xwin_version="0.9.0" && \
  xwin_prefix="xwin-$xwin_version-x86_64-unknown-linux-musl" && \
  curl --fail -L https://github.com/Jake-Shadle/xwin/releases/download/$xwin_version/$xwin_prefix.tar.gz | \
  tar -xzv -C /usr/local/bin --strip-components=1 $xwin_prefix/xwin && \
  xwin --accept-license splat --include-debug-libs --include-debug-symbols --output /xwin && \
  rm -rf .xwin-cache

RUN ln -sf /xwin/sdk/include/um/DbgHelp.h /xwin/sdk/include/um/Dbghelp.h && \
  ln -sf /xwin/sdk/lib/um/x86_64/dbghelp.lib /xwin/sdk/lib/um/x86_64/DbgHelp.lib && \
  ln -sf /xwin/sdk/lib/um/x86_64/dwrite.lib /xwin/sdk/lib/um/x86_64/Dwrite.lib && \
  ln -sf /xwin/sdk/lib/um/x86_64/d2d1.lib /xwin/sdk/lib/um/x86_64/D2d1.lib && \
  ln -sf /xwin/sdk/lib/um/x86_64/dcomp.lib /xwin/sdk/lib/um/x86_64/DComp.lib

RUN DXGI_PATH=$(find /xwin -type f -iname "dxgi.h" | head -n 1) && \
  DXGI_DIR=$(dirname "$DXGI_PATH") && \
  ln -sf "$DXGI_PATH" "$DXGI_DIR/Dxgi.h"

ENV CL_FLAGS="-Wno-unused-command-line-argument -fuse-ld=lld-link /vctoolsdir /xwin/crt /winsdkdir /xwin/sdk"
ENV CFLAGS_x86_64_pc_windows_msvc="$CL_FLAGS" \
  CXXFLAGS_x86_64_pc_windows_msvc="$CL_FLAGS"

# debug linking
ENV LDFLAGS="/libpath:/xwin/crt/lib/x86_64 /libpath:/xwin/sdk/lib/um/x86_64 /libpath:/xwin/sdk/lib/ucrt/x86_64"
ENV LIB="/xwin/crt/lib/x86_64;/xwin/sdk/lib/um/x86_64;/xwin/sdk/lib/ucrt/x86_64"

# symlink for debug .lib files
RUN cd /xwin && \
  find . -type f -name "*.lib" | while read -r f; do \
  dir=$(dirname "$f"); \
  base=$(basename "$f" .lib); \
  upper_base=$(echo "$base" | tr '[:lower:]' '[:upper:]'); \
  if [ ! -f "$dir/$upper_base.lib" ]; then ln -sf "$(basename "$f")" "$dir/$upper_base.lib"; fi; \
  if [ ! -f "$dir/$upper_base.LIB" ]; then ln -sf "$(basename "$f")" "$dir/$upper_base.LIB"; fi; \
  done

CMD ["bash", "-c", "echo 'Windows container started.'"]