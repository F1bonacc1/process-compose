{ config, lib, pkgs, ... }:

pkgs.buildGoModule rec {
  pname = "process-compose";
  version = "0.17.0";
  src = ./.;
  ldflags = [ "-X main.version=v${version}" ];

  vendorSha256 = "RcxU68vn2RirbOtEGcVrKYmEFeQDg1eovlQKXRandNs=";

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
