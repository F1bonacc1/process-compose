{ buildGoModule, config, lib, pkgs, installShellFiles, date, commit }:

buildGoModule rec {
  pname = "process-compose";
  version = "0.29.1";
  pkg = "github.com/f1bonacc1/process-compose/src/config";

  src = ./.;
  ldflags = [
    "-X ${pkg}.Version=v${version}"
    "-X ${pkg}.Date=${date}"
    "-X ${pkg}.Commit=${commit}"
    "-s"
    "-w"
  ];

  nativeBuildInputs = [ installShellFiles ];

  vendorSha256 = "fL12Rx/0TF2jjciSHgfIDfrqdQxxm2JiGfgO3Dgz81M=";
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
