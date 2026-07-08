package controllers

import (
	spacesvc "github.com/cloudreve/Cloudreve/v4/service/shared_space"
	"github.com/gin-gonic/gin"
)

func CreateSharedSpace(c *gin.Context) {
	service := ParametersFromContext[*spacesvc.CreateSharedSpaceService](c, spacesvc.CreateSharedSpaceParamCtx{})
	resp, err := service.Create(c)
	if err != nil {
		c.JSON(200, ErrorResponse(err))
		return
	}
	c.JSON(200, resp)
}

func ListSharedSpaces(c *gin.Context) {
	service := ParametersFromContext[*spacesvc.ListSharedSpaceService](c, spacesvc.ListSharedSpaceParamCtx{})
	resp, err := service.List(c)
	if err != nil {
		c.JSON(200, ErrorResponse(err))
		return
	}
	c.JSON(200, resp)
}

func UpdateSharedSpace(c *gin.Context) {
	service := ParametersFromContext[*spacesvc.CreateSharedSpaceService](c, spacesvc.CreateSharedSpaceParamCtx{})
	resp, err := service.Update(c)
	if err != nil {
		c.JSON(200, ErrorResponse(err))
		return
	}
	c.JSON(200, resp)
}

func DeleteSharedSpace(c *gin.Context) {
	service := ParametersFromContext[*spacesvc.SpaceIDService](c, spacesvc.SpaceIDParamCtx{})
	resp, err := service.Delete(c)
	if err != nil {
		c.JSON(200, ErrorResponse(err))
		return
	}
	c.JSON(200, resp)
}

func ListSpaceMembers(c *gin.Context) {
	service := ParametersFromContext[*spacesvc.ListSharedSpaceService](c, spacesvc.ListSharedSpaceParamCtx{})
	resp, err := service.Members(c)
	if err != nil {
		c.JSON(200, ErrorResponse(err))
		return
	}
	c.JSON(200, resp)
}

func AddSpaceMember(c *gin.Context) {
	service := ParametersFromContext[*spacesvc.AddMemberService](c, spacesvc.AddMemberParamCtx{})
	resp, err := service.Add(c)
	if err != nil {
		c.JSON(200, ErrorResponse(err))
		return
	}
	c.JSON(200, resp)
}

func UpdateSpaceMember(c *gin.Context) {
	service := ParametersFromContext[*spacesvc.UpdateMemberService](c, spacesvc.UpdateMemberParamCtx{})
	resp, err := service.Update(c)
	if err != nil {
		c.JSON(200, ErrorResponse(err))
		return
	}
	c.JSON(200, resp)
}

func RemoveSpaceMember(c *gin.Context) {
	service := ParametersFromContext[*spacesvc.RemoveMemberService](c, spacesvc.RemoveMemberParamCtx{})
	resp, err := service.Remove(c)
	if err != nil {
		c.JSON(200, ErrorResponse(err))
		return
	}
	c.JSON(200, resp)
}
