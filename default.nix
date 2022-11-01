{ config, lib, pkgs, installShellFiles, ... }:

pkgs.buildGoModule rec {
  pname = "process-compose";
  version = "0.21.0";
  src = ./.;
  ldflags = [ "-X main.version=v${version}" ];

  nativeBuildInputs = [ installShellFiles ];

  vendorSha256 = "RqPH8gm8K8sLuRl4FTpGeitS++t3ygAgZ6OMvBCsCB8=";

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
