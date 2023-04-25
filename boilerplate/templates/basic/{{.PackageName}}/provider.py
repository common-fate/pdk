import typing
import provider
from provider import target, access
import structlog

log = structlog.get_logger()

class Provider(provider.Provider):
    """
    The Provider class contains initialization logic which your `grant` and `revoke` function relies upon.
    """

    # add configuration variables to your provider by uncommenting the below.
    # These variables will be specified by users when they deploy your Access Provider.

    # api_url = provider.String(description="The API URL")
    # api_key = provider.String(description="The API key", secret=True)

    def setup(self):
        # construct any API clients here

        # you can reference config values as follows:
        # url = self.api_url.get()

        pass



@access.target(kind="Environment")
class EnvironmentTarget:
    """
    Targets are the things that Access Providers grants access to.

    In this example, environment is a software development environment that the user can request access to.
    """

    environment = target.String(
        title="Software Development Environment",
    )

@access.grant()
def grant(p: Provider, subject: str, target: EnvironmentTarget) -> access.GrantResult:
    # you can remove these log messages - they're just there to show an example of how to write logs.
    log.info(f"granting access", subject=subject, target=target)

    # Add your grant logic here
    # You can reference Provider config values as follows:
    # p.api_url.get()


@access.revoke()
def revoke(p: Provider, subject: str, target: EnvironmentTarget):
    # you can remove these log messages - they're just there to show an example of how to write logs.
    log.info(f"revoking access", subject=subject, target=target)

    # Add your revoke logic here
    # You can reference Provider config values as follows:
    # p.api_url.get()

