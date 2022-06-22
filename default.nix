{ config, lib, pkgs, ... }:

pkgs.buildGoModule rec {
  pname = "process-compose";
  version = "0.10.2";
  src = ./.;
  ldflags = [ "-X main.version=${version}" ];

  vendorSha256 = "1syn7sfv2hqwyl16kg14rvgwil65zlabnv2g44bpi4q35xmv1q46";

  postInstall = "mv $out/bin/{src,process-compose}";

  meta = with lib; {
    description =
      "Process Compose is like docker-compose, but for orchestrating a suite of processes, not containers.";
    homepage = "https://github.com/F1bonacc1/process-compose";
    license = licenses.asl20;
    mainProgram = "process-compose";
  };
  doCheck = false; # it takes ages to run the tests
}
