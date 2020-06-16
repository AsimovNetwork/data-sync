package response

const (
	SystemError          = 1001
	BadArguments         = 1002
	NotEnoughAmount      = 1003
	DataNotExist         = 1004
	CallBlockChainFailed = 1005

	NoAccountFound         = 2001
	PublicKeyNotMatch      = 2002
	PublicKeyDecrytoError  = 2003
	GetTemplateListError   = 2004
	CreateTemplateError    = 2005
	TemplateExistenceError = 2006
	InstanceExistenceError = 2007
	CreateInstanceError    = 2008
	AuthValidateError      = 2009

	CreatePayfeeProposalError  = 2101
	GetPayfeeProposalListError = 2102
	PayPayfeeProposalError     = 2103
	RejectPayfeeProposalError  = 2104
	CreateMultiSigError        = 2121
	JoinMultiSigError          = 2122
	GetScriptWalletsError      = 2123
	CreateMasterSlaveError     = 2124
	JoinMasterSlaveError       = 2125
	InvalidToken               = 3001

	PrestigeNotEnoughError = 2200
	PowerNotEnoughError    = 2201
	SignUpRepeatedlyError  = 2202
	NonSignUpTimeError     = 2203
	QueryPowerError        = 2204

	NonOperateTimeError               = 2205
	OperateRepeatedlyError            = 2206
	PermissionDeniedError             = 2207
	MemberNotEnoughError              = 2208
	VoteTooManyChairmanError          = 2209
	VoteTooManyPermanentDirectorError = 2210
)

var errorInfo = map[uint16]string{
	SystemError:                       "system error",
	BadArguments:                      "bad arguments",
	NotEnoughAmount:                   "insufficient amount",
	NoAccountFound:                    "no account found",
	PublicKeyNotMatch:                 "public key is not match",
	PublicKeyDecrytoError:             "the public key can not decrypt the signed message",
	GetTemplateListError:              "get template list failed",
	CreateTemplateError:               "create contract template failed",
	AuthValidateError:                 "address auth validate error",
	InvalidToken:                      "InvalidToken",
	TemplateExistenceError:            "template is not exist",
	InstanceExistenceError:            "instance is not exist",
	CreateInstanceError:               "create instance failed",
	CreatePayfeeProposalError:         "create payfee proposal failed",
	GetPayfeeProposalListError:        "get payfee proposal list failed",
	PayPayfeeProposalError:            "B pay the proposal failed",
	RejectPayfeeProposalError:         "B reject the proposal failed",
	DataNotExist:                      "data does not exist",
	CreateMultiSigError:               "create multi sig failed",
	JoinMultiSigError:                 "join multi sig failed",
	GetScriptWalletsError:             "get script wallet error",
	CreateMasterSlaveError:            "create master slave failed",
	JoinMasterSlaveError:              "join master slave failed",
	PrestigeNotEnoughError:            "Prestige is not enough",
	PowerNotEnoughError:               "power is not enough",
	SignUpRepeatedlyError:             "can't sign up repeatedly",
	NonSignUpTimeError:                "non sign up time",
	QueryPowerError:                   "query power failed",
	NonOperateTimeError:               "non operate time error ",
	OperateRepeatedlyError:            "can't operate repeatedly",
	PermissionDeniedError:             "permission denied",
	MemberNotEnoughError:              "foundation member is not enough",
	VoteTooManyChairmanError:          "can't vote for more than one chairman",
	VoteTooManyPermanentDirectorError: "can't vote for more than five permanent directors",
	CallBlockChainFailed:              "call block chain failed",
}
