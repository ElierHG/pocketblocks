import { apps, auth, folders, settings } from "@/api";
import { mocker } from "@/mocker";
import { Application, Folder, Settings, FullUser } from "@/types";
import {
  authRoute,
  createAppList,
  createDefaultErrorResponse,
  createDefaultResponse,
  createFolderList,
  getAuthConfigs,
} from "@/utils";

const createResponseData = async (
  user: FullUser,
  systemSettings: Settings,
  folders: Folder[],
  apps: Application[],
) => {
  return {
    user: {
      id: user.id,
      createdBy: "anonymousId",
      name: user.name !== "NONAME" ? user.name : "Change Me",
      avatar: null,
      tpAvatarLink: null,
      state: "ACTIVATED",
      isEnabled: true,
      isAnonymous: false,
      connections: [
        {
          authId: "EMAIL",
          source: "EMAIL",
          name: user.name !== "NONAME" ? user.name : "Change Me",
          avatar: null,
          rawUserInfo: {
            email: user.email,
          },
          tokens: [],
        },
      ],
      hasSetNickname: true,
      orgTransformedUserInfo: null,
    },
    organization: {
      id: "ORG_ID",
      createdBy: "anonymousId",
      name: systemSettings.org_name,
      isAutoGeneratedOrganization: true,
      contactName: null,
      contactEmail: null,
      contactPhoneNumber: null,
      source: null,
      thirdPartyCompanyId: null,
      state: "ACTIVE",
      commonSettings: {
        themeList: systemSettings.themes,
        defaultHomePage: systemSettings.home_page,
        defaultTheme: systemSettings.theme,
        preloadCSS: systemSettings.css,
        preloadJavaScript: systemSettings.script,
        preloadLibs: systemSettings.libs,
      },
      logoUrl: systemSettings.logo,
      createTime: 0,
      authConfigs: await getAuthConfigs(),
    },
    folderInfoViews: await createFolderList(folders),
    homeApplicationViews: await createAppList(apps),
  };
};

// eslint-disable-next-line @typescript-eslint/no-explicit-any
async function showVerifiedMessage(user: FullUser, messageIns: any) {
  const urlParams = new URLSearchParams(window.location.search);
  const verifyToken = urlParams.get("verifyEmailToken");
  const authConfig = (await getAuthConfigs())[0];
  if (verifyToken) {
    const { status } = await auth.verifyEmailToken(verifyToken);
    messageIns.destroy();
    if (status === 200) {
      messageIns.info("Email verified!");
    } else {
      messageIns.error("Something went wrong!");
    }
  } else if (authConfig.customProps.type !== "username" && !user.verified) {
    messageIns.destroy();
    messageIns.info({
      content:
        "Access your email to verify your account. If you didn't receive an email, click here!",
      duration: 5,
      style: { cursor: "pointer" },
      onClick: () => {
        auth.sendVerifyEmail().then((response) => {
          messageIns.destroy();
          if (response.status === 200) {
            messageIns.info(
              "Email sent! Please visit your Mailbox and verify your account.",
            );
          } else {
            messageIns.error("Something went wrong!");
          }
        });
      },
    });
  }
}

export default [
  mocker.get(
    "/api/v1/applications/home",
    authRoute(async ({ messageIns }) => {
      const userResponse = await auth.getCurrentUser();
      const appsResponse = await apps.list();
      const foldersResponse = await folders.list();
      const settingsResponse = await settings.get();
      if (
        userResponse.data &&
        appsResponse.data &&
        foldersResponse.data &&
        settingsResponse.data
      ) {
        await showVerifiedMessage(userResponse.data, messageIns);
        return createDefaultResponse(
          await createResponseData(
            userResponse.data,
            settingsResponse.data,
            foldersResponse.data,
            appsResponse.data,
          ),
        );
      }
      return createDefaultErrorResponse([
        userResponse,
        appsResponse,
        foldersResponse,
        settingsResponse,
      ]);
    }),
  ),
];
