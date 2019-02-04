module TopBarTests exposing (all, userWithEmail, userWithId, userWithName, userWithUserName)

import Concourse
import Dict
import Effects
import Expect exposing (..)
import Routes
import Test exposing (..)
import TopBar exposing (userDisplayName)
import TopBar.Msgs


userWithId : Concourse.User
userWithId =
    { id = "some-id", email = "", name = "", userName = "", teams = Dict.empty }


userWithEmail : Concourse.User
userWithEmail =
    { id = "some-id", email = "some-email", name = "", userName = "", teams = Dict.empty }


userWithName : Concourse.User
userWithName =
    { id = "some-id", email = "some-email", name = "some-name", userName = "", teams = Dict.empty }


userWithUserName : Concourse.User
userWithUserName =
    { id = "some-id", email = "some-email", name = "some-name", userName = "some-user-name", teams = Dict.empty }


all : Test
all =
    describe "TopBar"
        [ describe "userDisplayName"
            [ test "displays user name if present" <|
                \_ ->
                    Expect.equal
                        "some-user-name"
                        (TopBar.userDisplayName userWithUserName)
            , test "displays name if no userName present" <|
                \_ ->
                    Expect.equal
                        "some-name"
                        (TopBar.userDisplayName userWithName)
            , test "displays email if no userName or name present" <|
                \_ ->
                    Expect.equal
                        "some-email"
                        (TopBar.userDisplayName userWithEmail)
            , test "clicking a pinned resource navigates to the pinned resource page" <|
                \_ ->
                    TopBar.init (Routes.Pipeline "team" "pipeline" [])
                        |> Tuple.first
                        |> TopBar.update (TopBar.Msgs.GoToPinnedResource (Routes.Resource "team" "pipeline" "resource" Nothing))
                        |> Tuple.second
                        |> Expect.equal [ Effects.NavigateTo "/teams/team/pipelines/pipeline/resources/resource" ]
            , test "displays id if no userName, name or email present" <|
                \_ ->
                    Expect.equal
                        "some-id"
                        (TopBar.userDisplayName userWithId)
            ]
        ]
