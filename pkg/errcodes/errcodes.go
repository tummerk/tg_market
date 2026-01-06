package errcodes

import "git.appkode.ru/pub/go/failure"

const (
	InternalServerError          failure.ErrorCode = "InternalServerError"
	TimeoutExceeded              failure.ErrorCode = "TimeoutExceeded"
	Forbidden                    failure.ErrorCode = "Forbidden"
	ValidationError              failure.ErrorCode = "ValidationError"
	AccessTokenExpired           failure.ErrorCode = "AccessTokenExpired"
	AccessTokenInvalid           failure.ErrorCode = "AccessTokenInvalid"
	RefreshTokenExpired          failure.ErrorCode = "RefreshTokenExpired" //nolint:gosec // false positive
	RefreshTokenInvalid          failure.ErrorCode = "RefreshTokenInvalid" //nolint:gosec // false positive
	NotFound                     failure.ErrorCode = "NotFound"
	CredentialsMismatch          failure.ErrorCode = "CredentialsMismatch"
	PhoneAlreadyInUse            failure.ErrorCode = "PhoneAlreadyInUse"
	EmployeeAlreadyInAnotherCrew failure.ErrorCode = "EmployeeAlreadyInAnotherCrew"
	CrewAlreadyHasMaster         failure.ErrorCode = "CrewAlreadyHasMaster"
	CrewAlreadyHasSeniorJanitor  failure.ErrorCode = "CrewAlreadyHasSeniorJanitor"
	OnlyMasterCanHaveFewCrews    failure.ErrorCode = "OnlyMasterCanHaveFewCrews"
	CrewNameAlreadyInUseInAgency failure.ErrorCode = "CrewNameAlreadyInUseInAgency"
	InvalidPhoneNumber           failure.ErrorCode = "InvalidPhoneNumber"
	InvalidCoordinates           failure.ErrorCode = "InvalidCoordinates"
	InvalidPushToken             failure.ErrorCode = "InvalidPushToken"
	InvalidUserStatus            failure.ErrorCode = "InvalidUserStatus"
	InvalidTaskStatus            failure.ErrorCode = "InvalidTaskStatus"
	InvalidTaskID                failure.ErrorCode = "InvalidTaskID"
	InvalidUserID                failure.ErrorCode = "InvalidUserID"
	InvalidUserRole              failure.ErrorCode = "InvalidUserRole"
	InvalidCleaningMode          failure.ErrorCode = "InvalidCleaningMode"
	InvalidURL                   failure.ErrorCode = "InvalidURL"
	InvalidRouteID               failure.ErrorCode = "InvalidRouteID"
	InvalidPasswordFormat        failure.ErrorCode = "InvalidPasswordFormat"
	InvalidPaging                failure.ErrorCode = "InvalidPaging"
	InvalidCrewID                failure.ErrorCode = "InvalidCrewID"
	InvalidCrewName              failure.ErrorCode = "InvalidCrewName"
	InvalidPhotoID               failure.ErrorCode = "InvalidPhotoID"
	InvalidPhotoURL              failure.ErrorCode = "InvalidPhotoURL"
	InvalidTaskTemplateID        failure.ErrorCode = "InvalidTaskTemplateID"
	InvalidAddress               failure.ErrorCode = "InvalidAddress"
	InvalidDescription           failure.ErrorCode = "InvalidDescription"
	InvalidExample               failure.ErrorCode = "InvalidExample"

	// Новые для Gift модуля
	GiftNotFound      failure.ErrorCode = "GiftNotFound"      // Когда ID есть, но в базе нет
	InvalidGiftID     failure.ErrorCode = "InvalidGiftID"     // Когда пришел мусор вместо ID
	GiftOutOfStock    failure.ErrorCode = "GiftOutOfStock"    // Закончился тираж
	InvalidStorePrice failure.ErrorCode = "InvalidStorePrice" // Цена
)
