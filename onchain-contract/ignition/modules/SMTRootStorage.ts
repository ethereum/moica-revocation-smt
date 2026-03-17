import { buildModule } from "@nomicfoundation/hardhat-ignition/modules";

export default buildModule("SMTRootStorageModule", (m) => {
  const relayer = m.getParameter("relayer");
  const smtRootStorage = m.contract("SMTRootStorage", [relayer]);
  return { smtRootStorage };
});
