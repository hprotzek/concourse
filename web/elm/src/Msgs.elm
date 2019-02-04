module Msgs exposing (Msg(..), NavIndex)

import Callback exposing (Callback)
import Effects
import Json.Encode
import Keyboard
import Routes
import SubPage.Msgs
import TopBar.Msgs


type alias NavIndex =
    Int


type Msg
    = RouteChanged Routes.Route
    | SubMsg NavIndex SubPage.Msgs.Msg
    | TopMsg NavIndex TopBar.Msgs.Msg
    | NewUrl String
    | ModifyUrl String
    | TokenReceived (Maybe String)
    | Callback Effects.LayoutDispatch Callback
    | KeyDown Keyboard.KeyCode
    | KeyUp Keyboard.KeyCode
    | ServerSentEvent Json.Encode.Value
