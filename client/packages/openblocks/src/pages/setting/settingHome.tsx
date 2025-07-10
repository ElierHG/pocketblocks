import { ThemeHome } from "./theme";
import { AdvancedSetting } from "./advanced/AdvancedSetting";
import { BrandingSettings } from "./branding/BrandingSetting";
import { IdSourceHome } from "./idSource"
import ConnectionsSetting from "./Connections"
import { trans } from "i18n";
import { TwoColumnSettingPageContent } from "./styled";
import SubSideBar from "components/layout/SubSideBar";
import { Menu } from "openblocks-design";
import history from "util/history";
import { useParams } from "react-router-dom";

enum SettingPageEnum {
  Member = "permission",
  Organization = "organization",
  Audit = "audit",
  Theme = "theme",
  Branding = "branding",
  Advanced = "advanced",
  IdSource = "idsource",
  Connections = "connections",
}

export function SettingHome() {
  const selectKey = useParams<{ setting: string }>().setting || SettingPageEnum.Member;

  const items = [
    {
      key: SettingPageEnum.Theme,
      label: trans("settings.theme"),
    },
    {
      key: SettingPageEnum.Advanced,
      label: trans("settings.advanced"),
    },
    {
      key: SettingPageEnum.Branding,
      label: trans("settings.branding"),
    },
    {
      key: SettingPageEnum.IdSource,
      label: trans("settings.idSource"),
    },
    {
      key: SettingPageEnum.Connections,
      label: trans("settings.connections"),
    },
  ];

  return (
    <TwoColumnSettingPageContent>
      <SubSideBar title={trans("settings.title")}>
        <Menu
          mode="inline"
          selectedKeys={[selectKey]}
          onClick={(value) => {
            history.push("/setting/" + value.key);
          }}
          items={items}
        />
      </SubSideBar>
      {selectKey === SettingPageEnum.Theme && <ThemeHome />}
      {selectKey === SettingPageEnum.Advanced && <AdvancedSetting />}
      {selectKey === SettingPageEnum.Branding && <BrandingSettings />}
      {selectKey === SettingPageEnum.IdSource && <IdSourceHome />}
      {selectKey === SettingPageEnum.Connections && <ConnectionsSetting />}
    </TwoColumnSettingPageContent>
  );
}

export default SettingHome;
