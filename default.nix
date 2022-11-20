{ buildGoModule, config, lib, pkgs, installShellFiles, version ? "latest" }:

buildGoModule {
  pname = "process-compose";
  version = version;
  src = ./.;
  ldflags = [ "-X main.version=v${version}" ];

  nativeBuildInputs = [ installShellFiles ];

  vendorSha256 = "IsO1B6z1/HoGQ8xdNKQqZ/eZd90WikDbU9XiP0z28mU=";

  postInstall = ''
    mv $out/bin/{src,process-compose}

    installShellCompletion --cmd process-compose \
      --bash <($out/bin/process-compose completion bash) \
      --zsh <($out/bin/process-compose completion zsh) \
      --fish <($out/bin/process-compose completion fish)
  '';

  meta = with lib; {
    description =
      "Process Compose is like docker-compose, but for orchestrating a suite of processes, not containers.";
    homepage = "https://github.com/F1bonacc1/process-compose";
    license = licenses.asl20;
    mainProgram = "process-compose";
  };

  doCheck = false; # it takes ages to run the tests
}
