import json
import logging

import schemas
from chalicelib.core import users, telemetry, tenants
from chalicelib.utils import captcha, smtp
from chalicelib.utils import helper
from chalicelib.utils import pg_client
from chalicelib.utils.TimeUTC import TimeUTC
from chalicelib.utils.password_hash import hash_password

logger = logging.getLogger(__name__)


async def create_tenant(data: schemas.UserSignupSchema):
    logger.info(f"==== Signup started at {TimeUTC.to_human_readable(TimeUTC.now())} UTC")
    errors = []
    if await tenants.tenants_exists():
        return {"errors": ["tenants already registered"]}

    email = data.email
    logger.debug(f"email: {email}")
    password = data.password.get_secret_value()
    fullname = data.fullname

    if email is None or len(email) < 5:
        errors.append("Invalid email address.")
    else:
        if users.email_exists(email):
            errors.append("Email address already in use.")
        if users.get_deleted_user_by_email(email) is not None:
            errors.append("Email address previously deleted.")

    if helper.allow_captcha() and not captcha.is_valid(data.g_recaptcha_response):
        errors.append("Invalid captcha.")

    if len(password) < 6:
        errors.append("Password is too short, it must be at least 6 characters long.")



    organization_name = data.organizationName
    if organization_name is None or len(organization_name) < 1:
        errors.append("Invalid organization name.")

    if len(errors) > 0:
        logger.warning(
            f"==> signup error for:\n email:{data.email}, fullname:{data.fullname}, organizationName:{data.organizationName}")
        logger.warning(errors)
        return {"errors": errors}

    project_name = "my first project"
    params = {
        "email": email, "password": password, "fullname": fullname, "projectName": project_name,
        "data": json.dumps({"lastAnnouncementView": TimeUTC.now()}), "organizationName": organization_name
    }
    query = f"""WITH t AS (
                    INSERT INTO public.tenants (name)
                        VALUES (%(organizationName)s)
                    RETURNING tenant_id, api_key
                ),
                 u AS (
                     INSERT INTO public.users (tenant_id, email, role, name, data)
                             VALUES ((SELECT tenant_id FROM t), %(email)s, 'owner', %(fullname)s,%(data)s)
                             RETURNING user_id,email,role,name
                 ),
                 au AS (
                    INSERT INTO public.basic_authentication (user_id, password)
                        VALUES ((SELECT user_id FROM u), %(hashed_password)s)
                )
                 INSERT INTO public.projects (tenant_id, name, active)
                 VALUES ((SELECT tenant_id FROM t), %(projectName)s, TRUE)
                 RETURNING project_id, (SELECT api_key FROM t) AS api_key;"""

    # Hash the password using Python instead of pgcrypto
    params['hashed_password'] = hash_password(password)
    
    with pg_client.PostgresClient() as cur:
        cur.execute(cur.mogrify(query, params))

    telemetry.new_client()
    r = users.authenticate(email, password)
    r["smtp"] = smtp.has_smtp()

    return {
        "jwt": r.pop("jwt"),
        "refreshToken": r.pop("refreshToken"),
        "refreshTokenMaxAge": r.pop("refreshTokenMaxAge"),
        "spotJwt": r.pop("spotJwt"),
        "spotRefreshToken": r.pop("spotRefreshToken"),
        "spotRefreshTokenMaxAge": r.pop("spotRefreshTokenMaxAge"),
        'data': {
            "scopeState": 2,
            "user": r
        }
    }
