{ buildGoModule, config, lib, pkgs, installShellFiles, date, commit }:

let pkg = "github.com/f1bonacc1/process-compose/src/config";
in
buildGoModule rec {
  pname = "process-compose";
  version = "1.64.1";
  go = pkgs.go_1_23;
  env.CGO_ENABLED = 0;

  src = lib.cleanSource ./.;
  ldflags = [
    "-X ${pkg}.Version=v${version}"
    "-X ${pkg}.Date=${date}"
    "-X ${pkg}.Commit=${commit}"
    "-s"
    "-w"
  ];

  nativeBuildInputs = [ installShellFiles ];

  vendorHash = "sha256-zdlpB+B5OXW5EAIYGiUF3E4M1Lek3/27794b7bfcREw=";
  #vendorHash = lib.fakeHash;

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
