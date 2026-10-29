package business

type RoleChange struct {
	RoleID          int
	RoleName        string
	UserCount       int64
	Description     string
	ConfirmDelaySec int
}

func DescribeRoleChange(roleID int, roleName string, userCount int64) RoleChange {
	return RoleChange{
		RoleID:          roleID,
		RoleName:        roleName,
		UserCount:       userCount,
		Description:     "Изменение затронет",
		ConfirmDelaySec: 5,
	}
}