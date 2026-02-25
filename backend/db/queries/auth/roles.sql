-- name: GetUserRoles :many
SELECT r.* FROM auth.roles r
JOIN auth.user_roles ur ON ur.role_id = r.role_id
WHERE ur.user_id = $1
ORDER BY r.role_name;

-- name: GetActiveFeatureFlags :many
SELECT DISTINCT f.flag_name
FROM auth.feature_flags f
WHERE (
    f.flag_default_enabled = true
    OR EXISTS (
        SELECT 1 FROM auth.role_feature_flags rff
        JOIN auth.user_roles ur ON ur.role_id = rff.role_id
        WHERE rff.flag_id = f.flag_id AND ur.user_id = $1
    )
    OR EXISTS (
        SELECT 1 FROM auth.user_feature_flags uff
        WHERE uff.flag_id = f.flag_id AND uff.user_id = $1 AND uff.user_flag_enabled = true
    )
)
AND NOT EXISTS (
    SELECT 1 FROM auth.user_feature_flags uff
    WHERE uff.flag_id = f.flag_id AND uff.user_id = $1 AND uff.user_flag_enabled = false
);

-- name: HasFeatureFlag :one
SELECT EXISTS (
    SELECT 1 FROM auth.feature_flags f
    WHERE f.flag_name = $2
    AND (
        f.flag_default_enabled = true
        OR EXISTS (
            SELECT 1 FROM auth.role_feature_flags rff
            JOIN auth.user_roles ur ON ur.role_id = rff.role_id
            WHERE rff.flag_id = f.flag_id AND ur.user_id = $1
        )
        OR EXISTS (
            SELECT 1 FROM auth.user_feature_flags uff
            WHERE uff.flag_id = f.flag_id AND uff.user_id = $1 AND uff.user_flag_enabled = true
        )
    )
    AND NOT EXISTS (
        SELECT 1 FROM auth.user_feature_flags uff
        WHERE uff.flag_id = f.flag_id AND uff.user_id = $1 AND uff.user_flag_enabled = false
    )
) AS has_flag;

-- name: HasRole :one
SELECT EXISTS (
    SELECT 1 FROM auth.user_roles ur
    JOIN auth.roles r ON r.role_id = ur.role_id
    WHERE ur.user_id = $1 AND r.role_name = $2
) AS has_role;
