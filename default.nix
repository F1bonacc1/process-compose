{ buildGoModule, config, lib, pkgs, installShellFiles, version ? "latest" }:

buildGoModule {
  pname = "process-compose";
  version = version;
  src = ./.;
  ldflags = [ "-X github.com/f1bonacc1/process-compose/src/config.Version=v${version} -s -w" ];

  nativeBuildInputs = [ installShellFiles ];

  vendorSha256 = "CV/EEvATpjt1TATSrVIGzM7rOC0GXDEd8lwoRN3B2rQ=";

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
