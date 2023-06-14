{ buildGoModule, config, lib, pkgs, installShellFiles, date, commit }:

let pkg = "github.com/f1bonacc1/process-compose/src/config";
in
buildGoModule rec {
  pname = "process-compose";
  version = "0.51.0";


  src = lib.cleanSource ./.;
  ldflags = [
    "-X ${pkg}.Version=v${version}"
    "-X ${pkg}.Date=${date}"
    "-X ${pkg}.Commit=${commit}"
    "-s"
    "-w"
  ];

  nativeBuildInputs = [ installShellFiles ];

  vendorSha256 = "dlTqBKyI2t3twxQ+mnn+LTWzM2+CnEa4X0K2yDAZsQA=";
  #vendorSha256 = lib.fakeSha256;

  postInstall = ''
    mv $out/bin/{src,process-compose}

    installShellCompletion --cmd process-compose \
      --bash <($out/bin/process-compose completion bash) \
      --zsh <($out/bin/process-compose completion zsh) \
      --fish <($out/bin/process-compose completion fish)
  '';

  meta = with lib; {
    description = "A simple and flexible scheduler and orchestrator to manage non-containerized applications";
    homepage = "https://github.com/F1bonacc1/process-compose";
    changelog = "https://github.com/F1bonacc1/process-compose/releases/tag/v${version}";
    license = licenses.asl20;
    mainProgram = "process-compose";
  };

  doCheck = false; # it takes ages to run the tests
}
